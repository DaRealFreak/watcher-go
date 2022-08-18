// Package deviantart contains the implementation of the deviantart module
package deviantart

import (
	"fmt"
	"regexp"

	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/modules/deviantart/api"
	"github.com/DaRealFreak/watcher-go/internal/modules/deviantart/napi"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// deviantArt contains the implementation of the ModuleInterface
type deviantArt struct {
	*models.Module
	daAPI     *api.DeviantartAPI
	nAPI      *napi.DeviantartNAPI
	daPattern deviantArtPattern
	settings  deviantArtSettings
}

type deviantArtSettings struct {
	Download struct {
		DescriptionMinLength  int  `mapstructure:"description_min_length"`
		FollowForContent      bool `mapstructure:"follow_for_content"`
		UnfollowAfterDownload bool `mapstructure:"unfollow_after_download"`
	} `mapstructure:"download"`
	Cloudflare struct {
		UserAgent string `mapstructure:"user_agent"`
	} `mapstructure:"cloudflare"`
	UseDevAPI bool `mapstructure:"use_dev_api"`
}

type deviantArtPattern struct {
	feedPattern           *regexp.Regexp
	userPattern           *regexp.Regexp
	galleryPattern        *regexp.Regexp
	collectionUUIDPattern *regexp.Regexp
	collectionPattern     *regexp.Regexp
	tagPattern            *regexp.Regexp
	searchPattern         *regexp.Regexp
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
func (m *deviantArt) InitializeModule() {
	// initialize settings
	raven.CheckError(viper.UnmarshalKey(
		fmt.Sprintf("Modules.%s", m.GetViperModuleKey()),
		&m.settings,
	))
}

// AddModuleCommand adds custom module specific settings and commands to our application
func (m *deviantArt) AddModuleCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
}

// Login logs us in for the current session if possible/account available
func (m *deviantArt) Login(account *models.Account) bool {
	if m.settings.UseDevAPI {
		m.daAPI = api.NewDeviantartAPI(m.Key, account)

		usedProxy := m.GetProxySettings()
		if usedProxy.Enable {
			if err := m.daAPI.UserSession.SetProxy(usedProxy); err != nil {
				return false
			}

			if err := m.daAPI.Session.SetProxy(usedProxy); err != nil {
				return false
			}
		}

		m.daAPI.AddRoundTrippers(m.settings.Cloudflare.UserAgent)

		res, err := m.daAPI.Placebo()
		m.LoggedIn = err == nil && res.Status == "success"
	} else {
		m.nAPI = napi.NewDeviantartNAPI(m.Key)
		usedProxy := m.GetProxySettings()
		if usedProxy.Enable {
			if err := m.nAPI.UserSession.SetProxy(usedProxy); err != nil {
				return false
			}
		}

		m.nAPI.AddRoundTrippers(m.settings.Cloudflare.UserAgent)
		err := m.nAPI.Login(account)
		m.LoggedIn = err == nil
	}

	return m.LoggedIn
}

// Parse parses the tracked item
func (m *deviantArt) Parse(item *models.TrackedItem) (err error) {
	switch {
	case m.daPattern.feedPattern.MatchString(item.URI):
		if m.settings.UseDevAPI {
			return m.parseFeedDevApi(item)
		} else {
			return m.parseFeedNapi(item)
		}
	case m.daPattern.userPattern.MatchString(item.URI):
		if m.settings.UseDevAPI {
			return m.parseUserDevAPI(item)
		} else {
			return m.parseUserNapi(item)
		}
	case m.daPattern.galleryPattern.MatchString(item.URI):
		if m.settings.UseDevAPI {
			return m.parseGalleryDevAPI(item)
		} else {
			return m.parseGalleryNapi(item)
		}
	case m.daPattern.collectionPattern.MatchString(item.URI):
		if m.settings.UseDevAPI {
			return m.parseCollectionDevAPI(item)
		} else {
			return m.parseCollectionNapi(item)
		}
	case m.daPattern.collectionUUIDPattern.MatchString(item.URI):
		if m.settings.UseDevAPI {
			return m.parseCollectionUUIDDevAPI(item)
		} else {
			return m.parseCollectionUUIDNapi(item)
		}
	case m.daPattern.tagPattern.MatchString(item.URI):
		if m.settings.UseDevAPI {
			return m.parseTagDevAPI(item)
		} else {
			return m.parseTagNapi(item)
		}
	case m.daPattern.searchPattern.MatchString(item.URI):
		if m.settings.UseDevAPI {
			return m.parseSearchDevAPI(item)
		} else {
			return m.parseSearchNapi(item)
		}
	default:
		return fmt.Errorf("URL could not be associated with any of the implemented methods")
	}
}

// getDeviantArtPattern returns all required patterns
// extracted from the NewBareModule function to test in Unit Tests
func getDeviantArtPattern() deviantArtPattern {
	return deviantArtPattern{
		userPattern:           regexp.MustCompile(`https://www.deviantart.com/([^/?&]+?)(?:/gallery|/gallery/all)?(?:/)?$`),
		feedPattern:           regexp.MustCompile(`https://www.deviantart.com(?:/)?$`),
		galleryPattern:        regexp.MustCompile(`https://www.deviantart.com/([^/?&]+?)/gallery/(\d+).*`),
		collectionPattern:     regexp.MustCompile(`https://www.deviantart.com/([^/?&]+?)/favourites(?:/(\d+))?.*`),
		collectionUUIDPattern: regexp.MustCompile(`DeviantArt://collection/([^/?&]+?)/([^/?&]+)`),
		tagPattern:            regexp.MustCompile(`https://www.deviantart.com/tag/([^/?&]+)(?:$|/.*)`),
		searchPattern:         regexp.MustCompile(`https://www.deviantart.com/search.*q=(.*)`),
	}
}
