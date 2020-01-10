// Package pixiv contains the implementation of the pixiv module
package pixiv

import (
	"fmt"
	"regexp"

	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	ajaxapi "github.com/DaRealFreak/watcher-go/internal/modules/pixiv/ajax_api"
	mobileapi "github.com/DaRealFreak/watcher-go/internal/modules/pixiv/mobile_api"
	publicapi "github.com/DaRealFreak/watcher-go/internal/modules/pixiv/public_api"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/DaRealFreak/watcher-go/pkg/imaging/animation"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	// SearchAPIPublic is the key constant for the public search API (no limitations)
	SearchAPIPublic = "public"
	//SearchAPIMobile is the key constant for the mobile search API (limited to 5000 results)
	SearchAPIMobile = "mobile"

	// Ugoira is the returned API type for animations
	Ugoira = "ugoira"
)

// pixiv contains the implementation of the ModuleInterface and custom required variables
type pixiv struct {
	*models.Module
	animationHelper *animation.Helper
	publicAPI       *publicapi.PublicAPI
	mobileAPI       *mobileapi.MobileAPI
	ajaxAPI         *ajaxapi.AjaxAPI
	patterns        pixivPattern
	settings        pixivSettings
}

type pixivSettings struct {
	SearchAPI string `mapstructure:"search_api"`
	Animation struct {
		Format                string `mapstructure:"format"`
		LowQualityGifFallback bool   `mapstructure:"fallback_gif"`
	} `mapstructure:"animation"`
}

type pixivPattern struct {
	searchPattern       *regexp.Regexp
	illustrationPattern *regexp.Regexp
	fanboxPattern       *regexp.Regexp
	memberPattern       *regexp.Regexp
}

// nolint: gochecknoinits
// init function registers the bare and the normal module to the module factories
func init() {
	modules.GetModuleFactory().RegisterModule(NewBareModule())
}

// NewBareModule returns a bare module implementation for the CLI options
func NewBareModule() *models.Module {
	module := &models.Module{
		Key:           "pixiv.net",
		RequiresLogin: true,
		LoggedIn:      false,
		URISchemas: []*regexp.Regexp{
			regexp.MustCompile(".*pixiv.(co.jp|net)"),
		},
	}
	module.ModuleInterface = &pixiv{
		Module:          module,
		animationHelper: animation.NewAnimationHelper(),
		patterns: pixivPattern{
			searchPattern:       regexp.MustCompile(`(?:/tags/([^/?&]*)?|/search.php.*word=([^/?&]*)?)`),
			illustrationPattern: regexp.MustCompile(`(?:/artworks/(\d*)?|/member_illust.php?.*illust_id=(\d*)?)`),
			fanboxPattern:       regexp.MustCompile(`/fanbox/creator/(\d*)?`),
			memberPattern:       regexp.MustCompile(`(?:/member.php?.*id=(\d*)?|/member_illust.php?.*id=(\d*)?)`),
		},
	}

	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key,
		Color: "232:31",
	})

	return module
}

// InitializeModule initializes the module
func (m *pixiv) InitializeModule() {
	// initialize settings
	raven.CheckError(viper.UnmarshalKey(
		fmt.Sprintf("Modules.%s", m.GetViperModuleKey()),
		&m.settings,
	))

	if m.settings.Animation.Format == "" || !map[string]bool{
		animation.FileFormatGif:  true,
		animation.FileFormatWebp: true,
	}[m.settings.Animation.Format] {
		m.settings.Animation.Format = animation.FileFormatWebp
	}

	if m.settings.SearchAPI == "" {
		m.settings.SearchAPI = SearchAPIPublic
	}
}

// AddSettingsCommand adds custom module specific settings and commands to our application
func (m *pixiv) AddSettingsCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
}

// Login logs us in for the current session if possible/account available
func (m *pixiv) Login(account *models.Account) bool {
	m.mobileAPI = mobileapi.NewMobileAPI(m.Key, account)
	m.publicAPI = publicapi.NewPublicAPI(m.Key, account)

	raven.CheckError(m.preparePixivAPISessions())

	m.LoggedIn = true

	return m.LoggedIn
}

// Parse parses the tracked item
func (m *pixiv) Parse(item *models.TrackedItem) (err error) {
	switch {
	case m.patterns.searchPattern.MatchString(item.URI):
		switch m.settings.SearchAPI {
		case SearchAPIPublic:
			return m.parseSearchPublic(item)
		case SearchAPIMobile:
			return m.parseSearch(item)
		}
	case m.patterns.illustrationPattern.MatchString(item.URI):
		err := m.parseIllustration(item)
		if err == nil {
			m.DbIO.ChangeTrackedItemCompleteStatus(item, true)
		}

		return err
	case m.patterns.fanboxPattern.MatchString(item.URI):
		if m.ajaxAPI == nil {
			m.ajaxAPI = ajaxapi.NewAjaxAPI(m.Key)
			m.ajaxAPI.SessionCookie = m.DbIO.GetCookie(ajaxapi.CookieSession, m)

			if err := m.preparePixivAjaxSession(); err != nil {
				return err
			}
		}

		return m.parseFanbox(item)
	case m.patterns.memberPattern.MatchString(item.URI):
		return m.parseUser(item)
	default:
		return fmt.Errorf("URL could not be associated with any of the implemented methods")
	}

	return nil
}

func (m *pixiv) preparePixivAPISessions() error {
	usedProxy := m.GetProxySettings()

	// set proxy and overwrite the client if the proxy is enabled
	if usedProxy != nil && usedProxy.Enable {
		if err := m.publicAPI.Session.SetProxy(usedProxy); err != nil {
			return err
		}

		if err := m.mobileAPI.Session.SetProxy(usedProxy); err != nil {
			return err
		}
	}

	// prepare public API session again
	if err := m.publicAPI.AddRoundTrippers(); err != nil {
		return err
	}

	// prepare mobile API session again
	if err := m.mobileAPI.AddRoundTrippers(); err != nil {
		return err
	}

	return nil
}

func (m *pixiv) preparePixivAjaxSession() error {
	usedProxy := m.GetProxySettings()

	if usedProxy != nil && usedProxy.Enable {
		if err := m.ajaxAPI.Session.SetProxy(usedProxy); err != nil {
			return err
		}
	}

	m.ajaxAPI.AddRoundTrippers()

	return nil
}
