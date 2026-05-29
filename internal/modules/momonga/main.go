// Package momonga contains the implementation of the momon-ga.com module
package momonga

import (
	"fmt"
	"regexp"
	"sync"
	"time"

	formatter "github.com/DaRealFreak/colored-nested-formatter/v2"
	"github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/http/tls_session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/time/rate"
)

// momonga contains the implementation of the ModuleInterface
type momonga struct {
	*models.Module
	rateLimit      int
	galleryPattern *regexp.Regexp
	imagePattern   *regexp.Regexp
	settings       momongaSettings
	proxies        []*proxySession
	multiProxy     struct {
		currentIndexes []int
		waitGroup      sync.WaitGroup
		mutex          sync.Mutex
	}
}

type momongaSettings struct {
	Loop        bool                 `mapstructure:"loop"`
	LoopProxies []http.ProxySettings `mapstructure:"loopproxies"`
	MultiProxy  bool                 `mapstructure:"multiproxy"`
	RateLimit   *int                 `mapstructure:"rate_limit"`
	Search      struct {
		BlacklistedTags  []string `mapstructure:"blacklisted_tags"`
		CategorizeSearch bool     `mapstructure:"categorize_search"`
		InheritSubFolder bool     `mapstructure:"inherit_sub_folder"`
	} `mapstructure:"search"`
}

// nolint: gochecknoinits
// init function registers the bare module to the module factory
func init() {
	modules.GetModuleFactory().RegisterModule(NewBareModule())
}

// NewBareModule returns a bare module implementation for the CLI options
func NewBareModule() *models.Module {
	module := &models.Module{
		Key:           "momon-ga.com",
		RequiresLogin: false,
		LoggedIn:      false,
		URISchemas: []*regexp.Regexp{
			regexp.MustCompile(`momon-ga\.com`),
		},
		ProxyLoopIndex: -1,
		SettingsSchema: momongaSettings{},
	}
	module.ModuleInterface = &momonga{
		Module:         module,
		galleryPattern: regexp.MustCompile(`/(?:fanzine|magazine)/mo(\d+)`),
		imagePattern:   regexp.MustCompile(`/galleries/\d+/(\d+)\.\w+`),
	}

	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key,
		Color: "231:53",
	})

	return module
}

// InitializeModule initializes the module
func (m *momonga) InitializeModule() {
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

	momongaSession := tls_session.NewTlsClientSession(m.Key)
	momongaSession.RateLimiter = rate.NewLimiter(rate.Every(time.Duration(m.rateLimit)*time.Millisecond), 1)
	m.Session = momongaSession

	// set the proxy if requested
	raven.CheckError(m.setProxyMethod())

	// initialize the proxy sessions if multi-proxy is enabled
	if m.settings.MultiProxy {
		m.initializeProxySessions()
	}
}

// AddModuleCommand adds custom module specific settings and commands to our application
func (m *momonga) AddModuleCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
	m.AddProxyLoopCommands(command)
}

// Login logs us in for the current session if possible/account available
func (m *momonga) Login(_ *models.Account) bool {
	return true
}

// Parse parses the tracked item
func (m *momonga) Parse(item *models.TrackedItem) error {
	if m.galleryPattern.MatchString(item.URI) {
		return m.parseGallery(item)
	}

	return m.parseListing(item)
}

// setProxyMethod determines what proxy method is being used and sets/updates the proxy configuration
func (m *momonga) setProxyMethod() error {
	switch {
	case m.settings.Loop && len(m.settings.LoopProxies) < 2:
		return fmt.Errorf("you need to at least register 2 proxies to loop")
	case !m.settings.Loop && m.GetProxySettings() != nil && m.GetProxySettings().Enable:
		return m.Session.SetProxy(m.GetProxySettings())
	case m.settings.Loop:
		// reset proxy loop index if we reach the limit with the next iteration
		if m.ProxyLoopIndex+1 == len(m.settings.LoopProxies) {
			m.ProxyLoopIndex = -1
		}
		m.ProxyLoopIndex++

		for !m.settings.LoopProxies[m.ProxyLoopIndex].Enable {
			// skip to the next proxy if the current one is disabled
			m.ProxyLoopIndex++
			if m.ProxyLoopIndex+1 == len(m.settings.LoopProxies) {
				m.ProxyLoopIndex = -1
				break
			}
		}

		return m.Session.SetProxy(&m.settings.LoopProxies[m.ProxyLoopIndex])
	default:
		return nil
	}
}
