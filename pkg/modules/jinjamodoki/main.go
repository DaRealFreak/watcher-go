// jinjamodoki contains the implementation of the jinjamodoki module
package jinjamodoki

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"regexp"
	"time"

	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/pkg/http/session"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/modules"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/time/rate"
)

// jinjaModoki contains the implementation of the ModuleInterface
type jinjaModoki struct {
	*models.Module
	baseURL              *url.URL
	chapterUpdatePattern *regexp.Regexp
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
	moduleSession := session.NewSession(m.Key)
	moduleSession.RateLimiter = rate.NewLimiter(rate.Every(1*time.Second), 1)
	m.Session = moduleSession
	// set the proxy if requested
	raven.CheckError(m.Session.SetProxy(m.GetProxySettings()))

	client := m.Session.GetClient()
	client.Transport = m.SetReferer(client.Transport)
}

// AddSettingsCommand adds custom module specific settings and commands to our application
func (m *jinjaModoki) AddSettingsCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
}

// Login logs us in for the current session if possible/account available
func (m *jinjaModoki) Login(account *models.Account) bool {
	return m.LoggedIn
}

// Parse parses the tracked item
func (m *jinjaModoki) Parse(item *models.TrackedItem) error {
	res, err := m.Session.Get(item.URI)
	if err != nil {
		log.WithField("module", m.Key).Fatal(err)
	}

	test, err := ioutil.ReadAll(res.Body)
	fmt.Println(string(test))
	return nil
}
