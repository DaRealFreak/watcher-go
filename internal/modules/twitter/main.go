// Package twitter contains the implementation of the twitter module
package twitter

import (
	"fmt"
	"github.com/DaRealFreak/watcher-go/internal/modules/twitter/twitter_settings"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/modules/twitter/graphql_api"

	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/spf13/viper"

	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/modules/twitter/api"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// twitter contains the implementation of the ModuleInterface
type twitter struct {
	*models.Module
	twitterAPI          *api.TwitterAPI
	twitterGraphQlAPI   *graphql_api.TwitterGraphQlAPI
	normalizedUriRegexp *regexp.Regexp
	settings            twitter_settings.TwitterSettings
}

// nolint: gochecknoinits
// init function registers the bare and the normal module to the module factories
func init() {
	modules.GetModuleFactory().RegisterModule(NewBareModule())
}

// NewBareModule returns a bare module implementation for the CLI options
func NewBareModule() *models.Module {
	module := &models.Module{
		Key:           "twitter.com",
		RequiresLogin: false,
		LoggedIn:      true,
		URISchemas: []*regexp.Regexp{
			regexp.MustCompile(`.*twitter.com`),
			regexp.MustCompile(`twitter:(graphQL|api)/\d+/.*`),
		},
	}
	module.ModuleInterface = &twitter{
		Module:              module,
		normalizedUriRegexp: regexp.MustCompile(`twitter:(graphQL|api)/\d+/.*`),
	}

	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key,
		Color: "232:39",
	})

	return module
}

// InitializeModule initializes the module
func (m *twitter) InitializeModule() {
	// initialize settings
	raven.CheckError(viper.UnmarshalKey(
		fmt.Sprintf("Modules.%s", m.GetViperModuleKey()),
		&m.settings,
	))

	// already initialized
	if m.Session != nil || (m.twitterGraphQlAPI != nil && m.twitterGraphQlAPI.Session != nil) {
		return
	}

	if !m.settings.Api.UseGraphQlApi {
		oauthClient := m.DbIO.GetOAuthClient(m)
		if oauthClient == nil || oauthClient.ClientID == "" || oauthClient.ClientSecret == "" {
			log.WithField("module", m.Key).Fatalf(
				"module requires an OAuth2 consumer ID and token",
			)
			// log.Fatal will already exit with error code 1, so the exit is just for the IDE here
			os.Exit(1)
		}

		m.twitterAPI = api.NewTwitterAPI(m.Key, oauthClient)
	} else {
		m.twitterGraphQlAPI = graphql_api.NewTwitterAPI(m.ModuleKey(), m.settings)
		m.twitterGraphQlAPI.AddRoundTrippers()

		if cookie := m.DbIO.GetCookie(graphql_api.CookieAuth, m); cookie != nil {
			requestUrl, _ := url.Parse("https://twitter.com/")
			m.twitterGraphQlAPI.Session.GetClient().Jar.SetCookies(
				requestUrl,
				[]*http.Cookie{
					{
						Name:   "auth_token",
						Value:  cookie.Value,
						MaxAge: 0,
					},
				},
			)
		} else {
			// ToDo: guest cookie
		}
	}
}

// AddModuleCommand adds custom module specific settings and commands to our application
func (m *twitter) AddModuleCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
}

// Login logs us in for the current session if possible/account available
func (m *twitter) Login(_ *models.Account) bool {
	return m.LoggedIn
}

// Parse parses the tracked item
func (m *twitter) Parse(item *models.TrackedItem) error {
	if m.settings.ConvertNameToId && !m.normalizedUriRegexp.MatchString(item.URI) && !strings.Contains(item.URI, "/status/") {
		newUri, err := m.AddItem(item.URI)
		if err == nil {
			m.DbIO.ChangeTrackedItemUri(item, newUri)
		} else {
			log.WithField("module", item.Module).Warningf(
				"unable to convert screen name to ID for URI %s (%s)", item.URI, err.Error(),
			)
		}
	}

	if strings.Contains(item.URI, "/status/") {
		return m.parseStatus(item)
	} else {
		return m.parsePage(item)
	}
}

func (m *twitter) AddItem(uri string) (string, error) {
	uri = strings.ReplaceAll(uri, "mobile.twitter.com", "twitter.com")

	// we require the API to extract the twitter ID, so initialize the module if it's not initialized yet
	if m.Session == nil && (m.twitterGraphQlAPI == nil || m.twitterGraphQlAPI.Session == nil) {
		m.InitializeModule()
	}

	if m.settings.ConvertNameToId && !strings.Contains(uri, "/status/") {
		if match, err := regexp.MatchString(".*twitter.com", uri); err == nil && match {
			screenName, screenNameErr := m.extractScreenName(uri)
			if screenNameErr != nil {
				return uri, screenNameErr
			}

			log.WithField("module", m.Module.Key).Infof(
				"converting twitter username \"%s\"", screenName,
			)

			if m.settings.Api.UseGraphQlApi {
				userInformation, userErr := m.twitterGraphQlAPI.UserByUsername(screenName)
				if userErr != nil || userInformation == nil || len(userInformation.Data.User.Result.RestID.String()) == 0 {
					return uri, userErr
				}

				uri = fmt.Sprintf(
					"twitter:graphQL/%s/%s",
					userInformation.Data.User.Result.RestID.String(),
					userInformation.Data.User.Result.Legacy.ScreenName,
				)
			} else {
				userInformation, userErr := m.twitterAPI.UserByUsername(screenName)
				if userErr != nil || userInformation == nil || len(userInformation.Data.ID.String()) == 0 {
					return uri, userErr
				}

				uri = fmt.Sprintf(
					"twitter:api/%s/%s",
					userInformation.Data.ID.String(),
					userInformation.Data.Username,
				)
			}
		}
	}

	return uri, nil
}
