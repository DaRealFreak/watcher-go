// Package jinjamodoki contains the implementation of the jinjamodoki module
package jinjamodoki

import (
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"

	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/internal/http/session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// jinjaModoki contains the implementation of the ModuleInterface
type jinjaModoki struct {
	*models.Module
	baseURL        *url.URL
	defaultSession *session.DefaultSession
	settings       *models.ProxyLoopConfiguration
}

// nolint: gochecknoinits
// init function registers the bare and the normal module to the module factories
func init() {
	modules.GetModuleFactory().RegisterModule(NewBareModule())
}

// NewBareModule returns a bare module implementation for the CLI options
func NewBareModule() *models.Module {
	module := &models.Module{
		Key:           "jinja-modoki.com",
		RequiresLogin: false,
		LoggedIn:      false,
		URISchemas: []*regexp.Regexp{
			regexp.MustCompile("jinja-modoki.com"),
		},
		ProxyLoopIndex: -1,
	}
	module.ModuleInterface = &jinjaModoki{
		Module: module,
	}

	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key,
		Color: "232:231",
	})

	return module
}

// InitializeModule initializes the module
func (m *jinjaModoki) InitializeModule() {
	// initialize settings
	raven.CheckError(viper.UnmarshalKey(
		fmt.Sprintf("Modules.%s", m.GetViperModuleKey()),
		&m.settings,
	))

	m.baseURL, _ = url.Parse("https://gs-uploader.jinja-modoki.com/")

	moduleSession := session.NewSession(m.Key)
	m.defaultSession = moduleSession
	m.Session = m.defaultSession

	// set the proxy if requested
	raven.CheckError(m.setProxyMethod())

	// disable browsing access restrictions
	res, err := m.post("https://gs-uploader.jinja-modoki.com/upld-index.php?", url.Values{
		"mode":          {"complete"},
		"prev_mode":     {"top"},
		"item":          {"restriction"},
		"restriction[]": {"0", "1", "2", "3", "4"},
	})
	raven.CheckError(err)

	// blacklisted IPs can't set access, so throw a fatal log if that occurs
	content, _ := io.ReadAll(res.Body)
	if strings.Contains(string(content),
		"You are not allowed to change your access settings for browsing-restricted contents") {
		log.WithField("module", m.Key).Fatal(
			fmt.Errorf("could not change browsing restrictions, your IP is most likely blacklisted"),
		)
	}
}

// AddModuleCommand adds custom module specific settings and commands to our application
func (m *jinjaModoki) AddModuleCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
	m.AddProxyLoopCommands(command)
}

// Login logs us in for the current session if possible/account available
func (m *jinjaModoki) Login(_ *models.Account) bool {
	return m.LoggedIn
}

// Parse parses the tracked item
func (m *jinjaModoki) Parse(item *models.TrackedItem) error {
	return m.parsePage(item)
}

// setProxyMethod determines what proxy method is being used and sets/updates the proxy configuration
func (m *jinjaModoki) setProxyMethod() error {
	switch {
	case m.settings == nil:
	case m.settings.Loop && len(m.settings.LoopProxies) < 2:
		return fmt.Errorf("you need to at least register 2 proxies to loop")
	case !m.settings.Loop && m.settings.Proxy.Enable:
		if err := m.Session.SetProxy(&m.settings.Proxy); err != nil {
			return err
		}
	case m.settings.Loop:
		// reset proxy loop index if we reach the limit with the next iteration
		if m.ProxyLoopIndex+1 == len(m.settings.LoopProxies) {
			m.ProxyLoopIndex = -1
		}
		m.ProxyLoopIndex++

		if err := m.Session.SetProxy(&m.settings.LoopProxies[m.ProxyLoopIndex]); err != nil {
			return err
		}
	default:
	}

	return nil
}
