package vimeo

import (
	"fmt"
	"regexp"

	"github.com/DaRealFreak/watcher-go/pkg/fp"

	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/internal/http/session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/spf13/cobra"
)

type vimeo struct {
	*models.Module
	defaultVideoURLPattern *regexp.Regexp
	masterJsonPattern      *regexp.Regexp
}

// init function registers the bare and the normal module to the module factories
func init() {
	modules.GetModuleFactory().RegisterModule(NewBareModule())
}

// NewBareModule returns a bare module implementation for the CLI options
func NewBareModule() *models.Module {
	module := &models.Module{
		Key:           "vimeo.com",
		RequiresLogin: false,
		LoggedIn:      false,
		URISchemas: []*regexp.Regexp{
			regexp.MustCompile("vimeo.com"),
		},
	}
	module.ModuleInterface = &vimeo{
		Module:                 module,
		defaultVideoURLPattern: regexp.MustCompile(`https://vimeo.com/(\d+)(?:/)(\w+|$)`),
		masterJsonPattern:      regexp.MustCompile(`.*/master.json.*`),
	}

	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key,
		Color: "231:39",
	})

	return module
}

// InitializeModule initializes the module
func (m *vimeo) InitializeModule() {
	// set the module implementation for access to the session, database, etc
	m.Session = session.NewSession(m.Key)

	// set the proxy if requested
	raven.CheckError(m.Session.SetProxy(m.GetProxySettings()))
}

// AddModuleCommand adds custom module specific settings and commands to our application
func (m *vimeo) AddModuleCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
}

// Login logs us in for the current session if possible/account available
func (m *vimeo) Login(_ *models.Account) bool {
	return true
}

// Parse parses the tracked item
func (m *vimeo) Parse(item *models.TrackedItem) error {
	masterJsonURL := item.URI
	videoTitle := fp.SanitizePath(item.URI, false)
	if !m.masterJsonPattern.MatchString(masterJsonURL) {
		playerJson, err := m.getPlayerJSON(item)
		if err != nil {
			return err
		}

		masterJsonURL = playerJson.GetMasterJSONUrl()
		videoTitle = fmt.Sprintf(
			"%s_%s_%s",
			playerJson.Video.ID.String(),
			playerJson.Video.Owner.Name,
			fp.SanitizePath(playerJson.GetVideoTitle(), false),
		)
	}

	return m.parseVideo(item, masterJsonURL, videoTitle)
}
