// Package schalenetwork contains the implementation of the Schale Network module
package schalenetwork

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

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
)

type schaleNetwork struct {
	*models.Module
	galleryPattern *regexp.Regexp
	tagPattern     *regexp.Regexp
	settings       schaleNetworkSettings
	crt            string
	proxies        []*proxySession
	multiProxy     struct {
		currentIndexes []int
		waitGroup      sync.WaitGroup
	}
}

type schaleNetworkSettings struct {
	Crt        string `mapstructure:"crt"`
	Cloudflare struct {
		UserAgent string `mapstructure:"user_agent"`
	} `mapstructure:"cloudflare"`
	MultiProxy  bool                          `mapstructure:"multiproxy"`
	LoopProxies []watcherHttp.ProxySettings   `mapstructure:"loopproxies"`
	Search      struct {
		CategorizeSearch bool `mapstructure:"categorize_search"`
	} `mapstructure:"search"`
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

	session := tls_session.NewTlsClientSession(m.Key)

	// replace the default client with a Firefox 147 profile, reusing the same cookie jar
	client, _ := tls_client.NewHttpClient(tls_client.NewNoopLogger(),
		tls_client.WithTimeoutSeconds(30*60),
		tls_client.WithClientProfile(profiles.Firefox_147),
		tls_client.WithRandomTLSExtensionOrder(),
		tls_client.WithCookieJar(session.Jar),
	)
	session.SetClient(client)

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

// Parse parses the tracked item
func (m *schaleNetwork) Parse(item *models.TrackedItem) error {
	if strings.Contains(item.URI, "/g/") || strings.Contains(item.URI, "/reader/") {
		return m.parseGallery(item)
	}

	return m.parseSearch(item)
}
