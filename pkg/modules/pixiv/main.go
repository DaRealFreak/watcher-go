// Package pixiv contains the implementation of the pixiv module
package pixiv

import (
	"fmt"
	"regexp"

	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/pkg/imaging/animation"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/modules"
	ajaxapi "github.com/DaRealFreak/watcher-go/pkg/modules/pixiv/ajax_api"
	mobileapi "github.com/DaRealFreak/watcher-go/pkg/modules/pixiv/mobile_api"
	publicapi "github.com/DaRealFreak/watcher-go/pkg/modules/pixiv/public_api"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	// SearchAPIPublic is the key constant for the public search API (no limitations)
	SearchAPIPublic = "public"
	//SearchAPIMobile is the key constant for the mobile search API (limited to 5000 results)
	SearchAPIMobile = "mobile"
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
			searchPattern:       regexp.MustCompile(`(?:/tags/(\w*)?|/search.php.*word=(\w*)?)`),
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

	if m.settings.Animation.Format == "" {
		m.settings.Animation.Format = animation.FILE_FORMAT_WEBP
	}

	if m.settings.SearchAPI == "" {
		m.settings.SearchAPI = SearchAPIPublic
	}

	raven.CheckError(viper.WriteConfig())
}

// AddSettingsCommand adds custom module specific settings and commands to our application
func (m *pixiv) AddSettingsCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
}

// Login logs us in for the current session if possible/account available
func (m *pixiv) Login(account *models.Account) bool {
	var err error

	m.mobileAPI, err = mobileapi.NewMobileAPI(m.Key, account)
	if err != nil {
		return false
	}

	m.publicAPI, err = publicapi.NewPublicAPI(m.Key, account)
	if err != nil {
		return false
	}

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
		fmt.Println("parse illustration: " + item.URI)
		fmt.Println(m.patterns.illustrationPattern.FindStringSubmatch(item.URI))
	case m.patterns.fanboxPattern.MatchString(item.URI):
		fmt.Println("parse fanbox: " + item.URI)
		fmt.Println(m.patterns.fanboxPattern.FindStringSubmatch(item.URI))
	case m.patterns.memberPattern.MatchString(item.URI):
		fmt.Println("parse user: " + item.URI)
		fmt.Println(m.patterns.memberPattern.FindStringSubmatch(item.URI))
	default:
		return fmt.Errorf("URL could not be associated with any of the implemented methods")
	}

	return nil
}
