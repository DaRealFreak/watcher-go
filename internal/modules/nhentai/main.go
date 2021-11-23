// Package nhentai contains the implementation of the nhentai module
package nhentai

import (
	"net/url"
	"regexp"
	"strings"

	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/internal/http/session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/spf13/cobra"
)

type nhentai struct {
	*models.Module
	baseURL                *url.URL
	galleryIDPattern       *regexp.Regexp
	searchGalleryIDPattern *regexp.Regexp
	thumbToImageRegexp     *regexp.Regexp
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
	m.baseURL, _ = url.Parse("https://nhentai.net/")
	m.Session = session.NewSession(m.Key)

	// set the proxy if requested
	raven.CheckError(m.Session.SetProxy(m.GetProxySettings()))
}

// AddModuleCommand adds custom module specific settings and commands to our application
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

// getAbsoluteUri adds the base scheme and host since the site is using relative links
func (m *nhentai) getAbsoluteUri(uri string) string {
	parsedUri, _ := url.Parse(uri)
	parsedUri.Scheme = m.baseURL.Scheme
	parsedUri.Host = m.baseURL.Host

	return parsedUri.String()
}
