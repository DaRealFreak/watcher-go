// Package pixiv contains the implementation of the pixiv module
package pixiv

import (
	"fmt"
	"os"
	"regexp"

	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	fanboxapi "github.com/DaRealFreak/watcher-go/internal/modules/pixiv/fanbox_api"
	mobileapi "github.com/DaRealFreak/watcher-go/internal/modules/pixiv/mobile_api"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/DaRealFreak/watcher-go/pkg/imaging/animation"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	// Ugoira is the returned API type for animations
	Ugoira = "ugoira"
)

// pixiv contains the implementation of the ModuleInterface and custom required variables
type pixiv struct {
	*models.Module
	animationHelper *animation.Helper
	mobileAPI       *mobileapi.MobileAPI
	fanboxAPI       *fanboxapi.FanboxAPI
	patterns        pixivPattern
	settings        pixivSettings
}

type pixivSettings struct {
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
			regexp.MustCompile(".*fanbox.(cc)"),
		},
	}
	module.ModuleInterface = &pixiv{
		Module:          module,
		animationHelper: animation.NewAnimationHelper(),
		patterns: pixivPattern{
			searchPattern:       regexp.MustCompile(`(?:/tags/|/search.php.*word=)([^/?&]*)?`),
			illustrationPattern: regexp.MustCompile(`(?:/artworks/|/member_illust.php?.*illust_id=)(\d*)?`),
			fanboxPattern:       regexp.MustCompile(`(\w*)?.fanbox.cc`),
			memberPattern:       regexp.MustCompile(`(?:/member.php?.*id=|/member_illust.php?.*id=|/users/)(\d*)?`),
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
}

// AddModuleCommand adds custom module specific settings and commands to our application
func (m *pixiv) AddModuleCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
	m.addRunCommand(command)
}

// Login logs us in for the current session if possible/account available
func (m *pixiv) Login(_ *models.Account) bool {
	oauthClient := m.DbIO.GetOAuthClient(m)
	if oauthClient == nil || oauthClient.ClientID == "" || oauthClient.ClientSecret == "" {
		log.WithField("module", m.Key).Fatalf(
			"module requires an OAuth2 consumer ID and token",
		)
		// log.Fatal will already exit with error code 1, so the exit is just for the IDE here
		os.Exit(1)
	}

	m.mobileAPI = mobileapi.NewMobileAPI(m.Key, oauthClient)

	raven.CheckError(m.preparePixivAPISessions())

	m.LoggedIn = true

	return m.LoggedIn
}

// Parse parses the tracked item
func (m *pixiv) Parse(item *models.TrackedItem) (err error) {
	switch {
	case m.patterns.searchPattern.MatchString(item.URI):
		return m.parseSearch(item)
	case m.patterns.illustrationPattern.MatchString(item.URI):
		err = m.parseIllustration(item)
		if err == nil {
			m.DbIO.ChangeTrackedItemCompleteStatus(item, true)
		}

		return err
	case m.patterns.fanboxPattern.MatchString(item.URI):
		if m.fanboxAPI == nil {
			m.fanboxAPI = fanboxapi.NewFanboxAPI(m.Key)
			m.fanboxAPI.SessionCookie = m.DbIO.GetCookie(fanboxapi.CookieSession, m)

			if err = m.preparePixivFanboxSession(); err != nil {
				return err
			}
		}

		return m.parseFanbox(item)
	case m.patterns.memberPattern.MatchString(item.URI):
		return m.parseUser(item)
	default:
		return fmt.Errorf("URL could not be associated with any of the implemented methods")
	}
}

func (m *pixiv) preparePixivAPISessions() error {
	usedProxy := m.GetProxySettings()

	// set proxy and overwrite the client if the proxy is enabled
	if usedProxy != nil && usedProxy.Enable {
		if err := m.mobileAPI.Session.SetProxy(usedProxy); err != nil {
			return err
		}
	}
	// prepare mobile API session again
	if err := m.mobileAPI.AddRoundTrippers(); err != nil {
		return err
	}

	return nil
}

func (m *pixiv) preparePixivFanboxSession() error {
	usedProxy := m.GetProxySettings()

	if usedProxy != nil && usedProxy.Enable {
		if err := m.fanboxAPI.Session.SetProxy(usedProxy); err != nil {
			return err
		}
	}

	m.fanboxAPI.AddRoundTrippers()

	return nil
}

func (m *pixiv) addRunCommand(command *cobra.Command) {
	runCmd := &cobra.Command{
		Use:   "run [domains]",
		Short: "update items of only the passed domain (either pixiv.net or fanbox.cc)",
		Long:  "update all tracked items of the passed domains which can be either pixiv.net or fanbox.cc.",
		Run: func(cmd *cobra.Command, args []string) {
			m.InitializeModule()

			for _, domain := range args {
				trackedItems := m.DbIO.GetTrackedItemsByDomain(domain, false)
				for _, item := range trackedItems {
					if item.Module != m.ModuleKey() {
						continue
					}

					if err := m.Parse(item); err != nil {
						log.WithField("module", item.Module).Warningf(
							"error occurred parsing item %s (%s), skipping", item.URI, err.Error(),
						)
					}
				}
			}
		},
	}

	// add run command to parent command
	command.AddCommand(runCmd)
}
