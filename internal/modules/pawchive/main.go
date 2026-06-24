package pawchive

import (
	"fmt"
	"net/url"
	"regexp"
	"time"

	formatter "github.com/DaRealFreak/colored-nested-formatter/v2"
	"github.com/DaRealFreak/watcher-go/internal/http/tls_session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/modules/pawchive/api"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/time/rate"
)

const (
	baseURL  = "https://pawchive.st"
	fileHost = "https://file.pawchive.st"
)

type pawchive struct {
	*models.Module
	baseUrl  *url.URL
	api      *api.Client
	settings pawchiveSettings
}

type pawchiveSettings struct {
	ExternalURLs struct {
		DownloadExternalItems     bool `mapstructure:"download_external_items"`
		PrintExternalItems        bool `mapstructure:"print_external_items"`
		SkipErrorsForExternalURLs bool `mapstructure:"skip_errors_for_external_urls"`
	} `mapstructure:"external_urls"`
}

// init registers the bare module to the module factory.
func init() {
	modules.GetModuleFactory().RegisterModule(NewBareModule())
}

// NewBareModule returns a bare module implementation for the CLI options.
func NewBareModule() *models.Module {
	module := &models.Module{
		Key:           "pawchive.st",
		RequiresLogin: false,
		LoggedIn:      false,
		URISchemas: []*regexp.Regexp{
			regexp.MustCompile(`pawchive.st`),
		},
		SettingsSchema: pawchiveSettings{},
	}
	module.ModuleInterface = &pawchive{
		Module: module,
	}

	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key,
		Color: "232:208",
	})

	return module
}

// InitializeModule initializes the module.
func (m *pawchive) InitializeModule() {
	raven.CheckError(viper.UnmarshalKey(
		fmt.Sprintf("Modules.%s", m.GetViperModuleKey()),
		&m.settings,
	))

	pawchiveSession := tls_session.NewTlsClientSession(m.Key)
	pawchiveSession.RateLimiter = rate.NewLimiter(rate.Every(time.Duration(2500)*time.Millisecond), 1)
	m.Session = pawchiveSession

	raven.CheckError(m.Session.SetProxy(m.GetProxySettings()))

	m.baseUrl, _ = url.Parse(baseURL)
	m.api = api.NewClient(baseURL, m.Session)
}

// AddModuleCommand adds custom module specific settings and commands.
func (m *pawchive) AddModuleCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
}

// Login logs us in for the current session if possible/account available.
func (m *pawchive) Login(_ *models.Account) bool {
	return true
}

func (m *pawchive) getSubFolder(item *models.TrackedItem) string {
	if item.SubFolder != "" {
		return item.SubFolder
	}

	search := regexp.MustCompile(`https://pawchive\.st/([^/?&]+)/user/([^/?&]+)`).FindStringSubmatch(item.URI)
	if len(search) == 3 {
		return fp.SanitizePath(fmt.Sprintf("%s/%s", search[1], search[2]), true)
	}

	return ""
}
