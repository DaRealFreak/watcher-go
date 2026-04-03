// Package schalenetwork contains the implementation of the Schale Network module
package schalenetwork

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"

	formatter "github.com/DaRealFreak/colored-nested-formatter/v2"
	watcherHttp "github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/http/tls_session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/time/rate"
)

type schaleNetwork struct {
	*models.Module
	galleryPattern *regexp.Regexp
	tagPattern     *regexp.Regexp
	settings       schaleNetworkSettings
	rateLimit      int
	crt            string
	proxies        []*proxySession
	multiProxy     struct {
		currentIndexes []int
		waitGroup      sync.WaitGroup
	}
	currentSite        string // domain of the current tracked item (e.g. "niyaniya.moe", "hdoujin.org")
	clearanceValidated bool   // whether the clearance has been validated for the current crt
}

type schaleNetworkSettings struct {
	Crt        string `mapstructure:"crt"`
	Cloudflare struct {
		UserAgent string `mapstructure:"user_agent"`
	} `mapstructure:"cloudflare"`
	RateLimit   *int                        `mapstructure:"rate_limit"`
	MultiProxy  bool                        `mapstructure:"multiproxy"`
	LoopProxies []watcherHttp.ProxySettings `mapstructure:"loopproxies"`
}

// nolint: gochecknoinits
// init function registers the bare and the normal module to the module factories
func init() {
	modules.GetModuleFactory().RegisterModule(NewBareModule())
}

// NewBareModule returns a bare module implementation for the CLI options
func NewBareModule() *models.Module {
	module := &models.Module{
		Key:           "niyaniya.moe",
		RequiresLogin: false,
		LoggedIn:      false,
		URISchemas: []*regexp.Regexp{
			regexp.MustCompile(`.*niyaniya\.moe`),
			regexp.MustCompile(`.*shupogaki\.moe`),
			regexp.MustCompile(`.*koharu\.to`),
			regexp.MustCompile(`.*anchira\.to`),
			regexp.MustCompile(`.*seia\.to`),
			regexp.MustCompile(`.*hoshino\.one`),
			regexp.MustCompile(`.*hdoujin\.org`),
		},
		ProxyLoopIndex: -1,
		SettingsSchema: schaleNetworkSettings{},
	}
	module.ModuleInterface = &schaleNetwork{
		Module:         module,
		galleryPattern: regexp.MustCompile(`/(?:g|reader)/(\d+)/([a-zA-Z0-9]+)`),
		tagPattern:     regexp.MustCompile(`/tag/([^/?#]+)`),
	}

	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key,
		Color: "232:81",
	})

	return module
}

// InitializeModule initializes the module
func (m *schaleNetwork) InitializeModule() {
	// initialize settings
	raven.CheckError(viper.UnmarshalKey(
		fmt.Sprintf("Modules.%s", m.GetViperModuleKey()),
		&m.settings,
	))

	if m.settings.RateLimit != nil {
		m.rateLimit = *m.settings.RateLimit
	} else {
		m.rateLimit = 2500
	}

	errorHandler := schaleErrorHandler{module: m}
	session := tls_session.NewTlsClientSession(m.Key, errorHandler)

	// replace the default client with a Firefox 147 profile, reusing the same cookie jar
	client, _ := tls_client.NewHttpClient(tls_client.NewNoopLogger(),
		tls_client.WithTimeoutSeconds(30*60),
		tls_client.WithClientProfile(profiles.Firefox_147),
		tls_client.WithRandomTLSExtensionOrder(),
		tls_client.WithCookieJar(session.Jar),
	)
	session.SetClient(client)
	session.RateLimiter = rate.NewLimiter(rate.Every(time.Duration(m.rateLimit)*time.Millisecond), 1)

	m.Session = session

	// set the proxy if requested
	raven.CheckError(m.Session.SetProxy(m.GetProxySettings()))

	// initialize multi-proxy sessions if enabled
	if m.settings.MultiProxy {
		m.initializeProxySessions()
	}
}

// AddModuleCommand adds custom module-specific settings and commands to our application
func (m *schaleNetwork) AddModuleCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
	m.AddProxyLoopCommands(command)
	m.addCrtCommand(command)
}

// addCrtCommand adds the "crt" subcommand to interactively set the clearance token
func (m *schaleNetwork) addCrtCommand(command *cobra.Command) {
	crtCmd := &cobra.Command{
		Use:   "crt",
		Short: "interactively set the clearance (crt) token",
		Long: "Sets the crt token required for downloading images.\n\n" +
			"To find the token:\n" +
			"  1. Open https://niyaniya.moe (or https://hdoujin.org) in your browser\n" +
			"  2. Open DevTools (F12)\n" +
			"  3. Go to Application -> Local Storage -> select the site\n" +
			"  4. Copy the value of the \"clearance\" key",
		Run: func(cmd *cobra.Command, args []string) {
			m.promptCrtRefresh()
		},
	}

	command.AddCommand(crtCmd)
}

// promptCrtRefresh interactively prompts the user for a new crt token and user agent, then saves them
func (m *schaleNetwork) promptCrtRefresh() {
	reader := bufio.NewReader(os.Stdin)

	slog.Info("open one of the following URLs in your browser: https://niyaniya.moe or https://hdoujin.org", "module", m.Key)
	slog.Info("then open DevTools (F12) -> Application -> Local Storage -> select the site and copy the \"clearance\" value", "module", m.Key)
	fmt.Print("Paste the crt token: ")

	crt, _ := reader.ReadString('\n')
	crt = strings.TrimFunc(crt, func(r rune) bool {
		return !unicode.IsGraphic(r)
	})
	crt = strings.TrimSpace(crt)

	if crt == "" {
		slog.Warn("no token provided, keeping old token", "module", m.Key)
		return
	}

	slog.Info("the crt token is tied to your browser's User-Agent, so it must match", "module", m.Key)
	slog.Info("find it in DevTools (F12) -> Console -> type: navigator.userAgent", "module", m.Key)
	fmt.Printf("Paste your browser User-Agent (leave empty to keep current): ")

	ua, _ := reader.ReadString('\n')
	ua = strings.TrimFunc(ua, func(r rune) bool {
		return !unicode.IsGraphic(r)
	})
	ua = strings.TrimSpace(ua)

	moduleKey := m.GetViperModuleKey()

	m.crt = crt
	m.clearanceValidated = false
	viper.Set(fmt.Sprintf("Modules.%s.crt", moduleKey), crt)

	if ua != "" {
		m.settings.Cloudflare.UserAgent = ua
		viper.Set(fmt.Sprintf("Modules.%s.cloudflare.user_agent", moduleKey), ua)
	}

	raven.CheckError(viper.WriteConfig())
	slog.Info("crt token saved successfully", "module", m.Key)
}

// SetCookies loads stored cookies and the crt token from the database
func (m *schaleNetwork) SetCookies() {
	m.Module.SetCookies()

	// load crt from settings first
	m.crt = m.settings.Crt

	// fall back to cookie if not set in settings
	if m.crt == "" {
		if cookie := m.DbIO.GetCookie("crt", m); cookie != nil {
			m.crt = cookie.Value
		}
	}
}

// Login logs us in for the current session if possible/account available
func (m *schaleNetwork) Login(_ *models.Account) bool {
	return m.LoggedIn
}

// setSiteFromURI sets the current site domain based on the tracked item URI
func (m *schaleNetwork) setSiteFromURI(uri string) {
	if strings.Contains(uri, "hdoujin.org") {
		m.currentSite = "hdoujin.org"
	} else {
		m.currentSite = "niyaniya.moe"
	}
}

// apiBaseURL returns the API base URL for the current site
func (m *schaleNetwork) apiBaseURL() string {
	if m.currentSite == "hdoujin.org" {
		return "https://api.hdoujin.org"
	}

	return "https://api.schale.network"
}

// siteBaseURL returns the site base URL for the current site
func (m *schaleNetwork) siteBaseURL() string {
	return "https://" + m.currentSite
}

// authBaseURL returns the auth base URL for the current site
func (m *schaleNetwork) authBaseURL() string {
	if m.currentSite == "hdoujin.org" {
		return "https://auth.hdoujin.org"
	}

	return "https://auth.schale.network"
}

// secFetchSite returns the Sec-Fetch-Site header value for the current site
func (m *schaleNetwork) secFetchSite() string {
	if m.currentSite == "hdoujin.org" {
		return "same-site"
	}

	return "cross-site"
}

// Parse parses the tracked item
func (m *schaleNetwork) Parse(item *models.TrackedItem) error {
	m.setSiteFromURI(item.URI)

	if strings.Contains(item.URI, "/g/") || strings.Contains(item.URI, "/reader/") {
		return m.parseGallery(item)
	}

	return m.parseSearch(item)
}
