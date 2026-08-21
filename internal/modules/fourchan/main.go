// Package fourchan contains the implementation of the 4chan module
package fourchan

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/DaRealFreak/watcher-go/internal/http"
	"golang.org/x/time/rate"

	formatter "github.com/DaRealFreak/colored-nested-formatter/v2"
	"github.com/DaRealFreak/watcher-go/internal/http/tls_session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	fhttp "github.com/bogdanfinn/fhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// archiveCookieHost is the archive origin the module fetches its pages from. It is a
// separate origin from the module key host (4chan.org) and the one behind Cloudflare.
const archiveCookieHost = "desuarchive.org"

// fourChan contains the implementation of the ModuleInterface
type fourChan struct {
	*models.Module
	rateLimit     int
	threadPattern *regexp.Regexp
	settings      fourChanSettings
	proxies       []*proxySession
	// evictedProxies holds the loop proxies (keyed host:port) that returned a 403 during
	// page retrieval; they are skipped by rotation for the remainder of the run.
	evictedProxies map[string]bool
	multiProxy     struct {
		currentIndexes []int
		waitGroup      sync.WaitGroup
	}
}

type fourChanSettings struct {
	Loop        bool                 `mapstructure:"loop"`
	LoopProxies []http.ProxySettings `mapstructure:"loopproxies"`
	MultiProxy  bool                 `mapstructure:"multiproxy"`
	RateLimit   *int                 `mapstructure:"rate_limit"`
	Cloudflare  struct {
		// UserAgent has to match the browser the cf_clearance cookie got harvested from,
		// Cloudflare validates the cookie against it. Empty falls back to defaultUserAgent.
		UserAgent string `mapstructure:"user_agent"`
	} `mapstructure:"cloudflare"`
	Search struct {
		BlacklistedTags  []string `mapstructure:"blacklisted_tags"`
		CategorizeSearch bool     `mapstructure:"categorize_search"`
		InheritSubFolder bool     `mapstructure:"inherit_sub_folder"`
	} `mapstructure:"search"`
}

// init function registers the bare and the normal module to the module factories
func init() {
	modules.GetModuleFactory().RegisterModule(NewBareModule())
}

// NewBareModule returns a bare module implementation for the CLI options
func NewBareModule() *models.Module {
	module := &models.Module{
		Key:           "4chan.org",
		RequiresLogin: false,
		LoggedIn:      false,
		URISchemas: []*regexp.Regexp{
			regexp.MustCompile("4chan.org"),
			regexp.MustCompile("desuarchive.org"),
		},
		ProxyLoopIndex: -1,
		SettingsSchema: fourChanSettings{},
	}
	module.ModuleInterface = &fourChan{
		Module:         module,
		threadPattern:  regexp.MustCompile(`.*/(?P<BoardId>.*)/thread/(?P<ThreadID>.*)/`),
		evictedProxies: make(map[string]bool),
	}

	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key,
		Color: "231:88",
	})

	return module
}

// InitializeModule initializes the module
func (m *fourChan) InitializeModule() {
	// initialize settings
	raven.CheckError(viper.UnmarshalKey(
		fmt.Sprintf("Modules.%s", m.GetViperModuleKey()),
		&m.settings,
	))

	if m.settings.RateLimit != nil {
		m.rateLimit = *m.settings.RateLimit
	} else {
		// default rate limit of 1 request every 2.5 seconds
		m.rateLimit = 2500
	}

	// set the module implementation for access to the session, database, etc
	fourChanSession := tls_session.NewTlsClientSession(m.Key)
	// replace the client with one carrying the browser default headers, desuarchive is
	// behind Cloudflare and rejects the "Go-http-client/2.0" User-Agent fhttp sends when
	// a request brings none of its own
	fourChanSession.SetClient(m.newHttpClient(fourChanSession.Jar))
	fourChanSession.RateLimiter = rate.NewLimiter(rate.Every(time.Duration(m.rateLimit)*time.Millisecond), 1)
	m.Session = fourChanSession

	// set the proxy if requested
	raven.CheckError(m.setProxyMethod())

	m.initializeProxySessions()
}

// AddModuleCommand adds custom module specific settings and commands to our application
func (m *fourChan) AddModuleCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
	m.AddProxyLoopCommands(command)
}

// Login logs us in for the current session if possible/account available
func (m *fourChan) Login(_ *models.Account) bool {
	return true
}

// SetCookies loads the stored cookies of the module and applies them to the archive host
// on top of the module key host. The cookies table keys rows by module only, so the
// generic implementation sets everything for the module key host (4chan.org) alone -
// while the Cloudflare relevant cookies (cf_clearance, foolfuuka_*) belong to
// desuarchive.org and would never be sent.
//
// The main session shares its cookie jar with every multi-proxy download session (see
// initializeProxySessions), so setting them here propagates them to the whole pool.
func (m *fourChan) SetCookies() {
	m.Module.SetCookies()

	if m.Session == nil || m.DbIO == nil {
		return
	}

	archiveURL, err := url.Parse(fmt.Sprintf("https://%s", archiveCookieHost))
	if err != nil {
		return
	}

	cookies := m.DbIO.GetAllCookies(m.Module)
	if len(cookies) == 0 {
		return
	}

	sessionCookies := make([]*fhttp.Cookie, 0, len(cookies))
	for _, cookie := range cookies {
		sessionCookies = append(sessionCookies, &fhttp.Cookie{
			Name:   cookie.Name,
			Value:  cookie.Value,
			Domain: archiveURL.Host,
		})
	}

	m.Session.SetCookies(archiveURL, sessionCookies)
}

// Parse parses the tracked item
func (m *fourChan) Parse(item *models.TrackedItem) error {
	if strings.Contains(item.URI, "/thread/") {
		return m.parseThread(item)
	} else if strings.Contains(item.URI, "/search/") {
		return m.parseSearch(item)
	}

	return nil
}

// setProxyMethod determines what proxy method is being used and sets/updates the proxy configuration
func (m *fourChan) setProxyMethod() error {
	switch {
	case m.settings.Loop && len(m.settings.LoopProxies) < 2:
		return fmt.Errorf("you need to at least register 2 proxies to loop")
	case !m.settings.Loop && m.GetProxySettings() != nil && m.GetProxySettings().Enable:
		return m.Session.SetProxy(m.GetProxySettings())
	case m.settings.Loop:
		// advance to the next enabled, non-evicted proxy (wrapping around). proxies are
		// evicted for the rest of the run when they return a 403 during page retrieval.
		next := m.nextLiveProxyIndex()
		if next == -1 {
			return fmt.Errorf("no usable loop proxies remaining (all disabled or evicted after 403)")
		}
		m.ProxyLoopIndex = next

		return m.Session.SetProxy(&m.settings.LoopProxies[m.ProxyLoopIndex])
	default:
		return nil
	}
}
