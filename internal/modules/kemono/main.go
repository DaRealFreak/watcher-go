package kemono

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/internal/http/tls_session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/modules/kemono/api"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/time/rate"
)

type kemono struct {
	*models.Module
	baseUrl  *url.URL
	api      *api.Client
	settings kemonoSettings
}

type kemonoSettings struct {
	ExternalURLs struct {
		DownloadExternalItems     bool `mapstructure:"download_external_items"`
		PrintExternalItems        bool `mapstructure:"print_external_items"`
		SkipErrorsForExternalURLs bool `mapstructure:"skip_errors_for_external_urls"`
	} `mapstructure:"external_urls"`
}

// init function registers the bare and the normal module to the module factories
func init() {
	modules.GetModuleFactory().RegisterModule(NewBareModule())
}

// NewBareModule returns a bare module implementation for the CLI options
func NewBareModule() *models.Module {
	module := &models.Module{
		Key:           "kemono.su",
		RequiresLogin: false,
		LoggedIn:      false,
		URISchemas: []*regexp.Regexp{
			regexp.MustCompile(`kemono.su`),
			regexp.MustCompile(`kemono.cr`),
			regexp.MustCompile(`coomer.su`),
		},
	}
	module.ModuleInterface = &kemono{
		Module: module,
	}

	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key,
		Color: "232:208",
	})

	return module
}

// InitializeModule initializes the module
func (m *kemono) InitializeModule() {
	// initialize settings
	raven.CheckError(viper.UnmarshalKey(
		fmt.Sprintf("Modules.%s", m.GetViperModuleKey()),
		&m.settings,
	))

	// set the module implementation for access to the session, database, etc
	kemonoSession := tls_session.NewTlsClientSession(m.Key)
	kemonoSession.RateLimiter = rate.NewLimiter(rate.Every(time.Duration(2500)*time.Millisecond), 1)
	m.Session = kemonoSession

	// set the proxy if requested
	raven.CheckError(m.Session.SetProxy(m.GetProxySettings()))
}

// AddModuleCommand adds custom module specific settings and commands to our application
func (m *kemono) AddModuleCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
}

// Login logs us in for the current session if possible/account available
func (m *kemono) Login(_ *models.Account) bool {
	return true
}

// Parse parses the tracked item
func (m *kemono) Parse(item *models.TrackedItem) error {
	if item.SubFolder == "" {
		m.DbIO.ChangeTrackedItemSubFolder(item, m.getSubFolder(item))
	}

	if strings.Contains(item.URI, "coomer") {
		m.baseUrl, _ = url.Parse("https://coomer.st")
	} else {
		m.baseUrl, _ = url.Parse("https://kemono.cr")
	}
	m.api = api.NewClient(m.baseUrl.String(), m.Session)

	if regexp.MustCompile(`.*/post/.*`).MatchString(item.URI) {
		return m.parsePost(item)
	}

	return m.parseUser(item)
}

func (m *kemono) getSubFolder(item *models.TrackedItem) string {
	if item.SubFolder != "" {
		return item.SubFolder
	}

	search := regexp.MustCompile(`https://(?:kemono|coomer).\w+/([^/?&]+)/user/([^/?&]+)`).FindStringSubmatch(item.URI)
	if len(search) == 3 {
		return fp.SanitizePath(fmt.Sprintf("%s/%s", search[1], search[2]), true)
	}

	return ""
}
