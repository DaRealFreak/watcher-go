// Package nhentai contains the implementation of the nhentai module
package nhentai

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/internal/http/tls_session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type nhentai struct {
	*models.Module
	baseURL                *url.URL
	galleryIDPattern       *regexp.Regexp
	searchGalleryIDPattern *regexp.Regexp
	thumbToImageRegexp     *regexp.Regexp
	settings               nhentaiSettings
}

type nhentaiSettings struct {
	Cloudflare struct {
		UserAgent string `mapstructure:"user_agent"`
	} `mapstructure:"cloudflare"`
	Search struct {
		BlacklistedTags  []string `mapstructure:"blacklisted_tags"`
		CategorizeSearch bool     `mapstructure:"categorize_search"`
		InheritSubFolder bool     `mapstructure:"inherit_sub_folder"`
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
		Key:           "nhentai.net",
		RequiresLogin: false,
		LoggedIn:      false,
		URISchemas: []*regexp.Regexp{
			regexp.MustCompile(`.*nhentai.net`),
		},
		ProxyLoopIndex: -1,
	}
	module.ModuleInterface = &nhentai{
		Module:                 module,
		thumbToImageRegexp:     regexp.MustCompile(`(/galleries/[\d]+/.*)t(\..*)`),
		galleryIDPattern:       regexp.MustCompile(`/galleries/(\d+)/.*`),
		searchGalleryIDPattern: regexp.MustCompile(`/g/(\d+)/`),
	}

	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key,
		Color: "255:197",
	})

	return module
}

// InitializeModule initializes the module
func (m *nhentai) InitializeModule() {
	// initialize settings
	raven.CheckError(viper.UnmarshalKey(
		fmt.Sprintf("Modules.%s", m.GetViperModuleKey()),
		&m.settings,
	))

	m.baseURL, _ = url.Parse("https://nhentai.net/")
	m.Session = tls_session.NewTlsClientSession(m.Key)

	// set the proxy if requested
	raven.CheckError(m.Session.SetProxy(m.GetProxySettings()))
}

// AddModuleCommand adds custom module-specific settings and commands to our application
func (m *nhentai) AddModuleCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
	m.AddProxyLoopCommands(command)
}

// Login logs us in for the current session if possible/account available
func (m *nhentai) Login(account *models.Account) bool {
	return m.LoggedIn
}

// Parse parses the tracked item
func (m *nhentai) Parse(item *models.TrackedItem) error {
	if strings.Contains(item.URI, "/g/") {
		return m.parseGallery(item)
	} else {
		return m.parseSearch(item)
	}
}
