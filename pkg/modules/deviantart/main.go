// Package deviantart contains the implementation of the deviantart module
package deviantart

import (
	"regexp"

	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/modules"
	"github.com/DaRealFreak/watcher-go/pkg/modules/deviantart/api"
	"github.com/spf13/cobra"
)

// deviantArt contains the implementation of the ModuleInterface
type deviantArt struct {
	*models.Module
	daAPI     *api.DeviantartAPI
	daPattern deviantArtPattern
}

type deviantArtPattern struct {
	feedPattern       *regexp.Regexp
	galleryPattern    *regexp.Regexp
	collectionPattern *regexp.Regexp
	tagPattern        *regexp.Regexp
}

// nolint: gochecknoinits
// init function registers the bare and the normal module to the module factories
func init() {
	modules.GetModuleFactory().RegisterModule(NewBareModule())
}

// NewBareModule returns a bare module implementation for the CLI options
func NewBareModule() *models.Module {
	module := &models.Module{
		Key:           "deviantart.com",
		RequiresLogin: true,
		LoggedIn:      false,
		URISchemas: []*regexp.Regexp{
			regexp.MustCompile(".*deviantart.com"),
			regexp.MustCompile(`DeviantArt://.*`),
		},
	}
	module.ModuleInterface = &deviantArt{
		Module:    module,
		daPattern: getDeviantArtPattern(),
	}

	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key,
		Color: "232:34",
	})

	return module
}

// InitializeModule initializes the module
func (m *deviantArt) InitializeModule() {}

// AddSettingsCommand adds custom module specific settings and commands to our application
func (m *deviantArt) AddSettingsCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
}

// Login logs us in for the current session if possible/account available
func (m *deviantArt) Login(account *models.Account) bool {
	m.daAPI = api.NewDeviantartAPI(m.Key, account)
	m.daAPI.AddRoundTrippers()

	usedProxy := m.GetProxySettings()
	if usedProxy.Enable {
		if err := m.daAPI.Session.SetProxy(usedProxy); err != nil {
			return false
		}

		if err := m.daAPI.Session.SetProxy(usedProxy); err != nil {
			return false
		}
	}

	m.LoggedIn = true

	return m.LoggedIn
}

// Parse parses the tracked item
func (m *deviantArt) Parse(item *models.TrackedItem) (err error) {
	return nil
}

// getDeviantArtPattern returns all required patterns
// extracted from the NewBareModule function to test in Unit Tests
func getDeviantArtPattern() deviantArtPattern {
	return deviantArtPattern{
		feedPattern: regexp.MustCompile(`DeviantArt://watchfeed|https://www.deviantart.com(?:/$|$)`),
		galleryPattern: regexp.MustCompile(`DeviantArt://gallery/(.*)` +
			`|https://www.deviantart.com/([^/?&]*)?/gallery/(\d+).*`),
		collectionPattern: regexp.MustCompile(`DeviantArt://collection/(.*)` +
			`|https://www.deviantart.com/([^/?&]*)?/favourites/(\d+).*`),
		tagPattern: regexp.MustCompile(`DeviantArt://tag/(.*)|https://www.deviantart.com/tag/([^/?&]*)?`),
	}
}
