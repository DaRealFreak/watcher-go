// Package fourchan contains the implementation of the 4chan module
package fourchan

import (
	"fmt"
	"github.com/DaRealFreak/watcher-go/internal/http"
	"golang.org/x/time/rate"
	"regexp"
	"strings"
	"sync"
	"time"

	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/internal/http/session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// fourChan contains the implementation of the ModuleInterface
type fourChan struct {
	*models.Module
	rateLimit     int
	threadPattern *regexp.Regexp
	settings      *fourChanSettings
	proxies       []*proxySession
	multiProxy    struct {
		currentIndexes []int
		waitGroup      sync.WaitGroup
	}
}

type fourChanSettings struct {
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
	}
	module.ModuleInterface = &fourChan{
		Module:        module,
		threadPattern: regexp.MustCompile(`.*/(?P<BoardId>.*)/thread/(?P<ThreadID>.*)/`),
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
	fourChanSession := session.NewSession(m.Key)
	fourChanSession.RateLimiter = rate.NewLimiter(rate.Every(time.Duration(m.rateLimit)*time.Millisecond), 1)
	m.Session = fourChanSession

	// set the proxy if requested
	raven.CheckError(m.setProxyMethod())

	m.initializeProxySessions()
}

// AddModuleCommand adds custom module specific settings and commands to our application
func (m *fourChan) AddModuleCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
}

// Login logs us in for the current session if possible/account available
func (m *fourChan) Login(_ *models.Account) bool {
	return true
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
	case m.settings == nil:
		return nil
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

		return m.Session.SetProxy(&m.settings.LoopProxies[m.ProxyLoopIndex])
	default:
		return nil
	}
}
