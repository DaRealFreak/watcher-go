// Package pixiv contains the implementation of the pixiv module
package pixiv

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"regexp"
	"strings"

	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/pkg/animation"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/modules/pixiv/session"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// pixiv contains the implementation of the ModuleInterface and custom required variables
type pixiv struct {
	models.Module
	pixivSession    *session.PixivSession
	animationHelper *animation.Helper
}

// downloadQueueItem contains the required variables to download items
type downloadQueueItem struct {
	ItemID       string
	DownloadTag  string
	Illustration *illustration
}

// search options
//noinspection GoUnusedConst
const (
	// single image
	SearchFilterIllustration = "illust"
	// multiple images
	SearchFilterManga = "manga"
	// multiple images concatenated by javascript in the frontend looking like an animation
	SearchFilterUgoira = "ugoira"
	// novels/text
	SearchFilterNovel = "novels"
	// everything
	SearchFilterAll = ""

	// order of the search results
	SearchOrderDateAscending        = "date_asc"
	SearchOrderDateDescending       = "date_desc"
	SearchOrderPopularityAscending  = "popular_asc"
	SearchOrderPopularityDescending = "popular_desc"

	// search mode new API
	SearchModePartialTagMatch = "partial_match_for_tags"
	SearchModeExactTagMatch   = "exact_match_for_tags"
	SearchModeTitleAndCaption = "title_and_caption"

	// search mode previous API
	PublicAPISearchModePartialTagMatch = "tag"
	PublicAPISearchModeExactTagMatch   = "exact_tag"
	PublicAPISearchModeText            = "text"
	PublicAPISearchModeCaption         = "caption"

	PublicAPISearchFilterIllustration = "illustration"
	PublicAPISearchFilterManga        = "manga"
	PublicAPISearchFilterUgoira       = "ugoira"
)

// NewModule generates new module and registers the URI schema
func NewModule(dbIO models.DatabaseInterface, uriSchemas map[string][]*regexp.Regexp) *models.Module {
	// register empty sub module to point to
	var subModule = pixiv{
		animationHelper: animation.NewAnimationHelper(),
	}

	// initialize the Module with the session/database and login status
	module := models.Module{
		DbIO:            dbIO,
		LoggedIn:        false,
		ModuleInterface: &subModule,
	}
	// set the module implementation for access to the session, database, etc
	subModule.Module = module
	subModule.pixivSession = session.NewSession(subModule.GetProxySettings())
	subModule.pixivSession.Module = &subModule
	subModule.pixivSession.ModuleKey = subModule.Key()
	subModule.Session = subModule.pixivSession

	// register the uri schema
	module.RegisterURISchema(uriSchemas)
	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key(),
		Color: "232:31",
	})

	return &module
}

// Key returns the module key
func (m *pixiv) Key() (key string) {
	return "pixiv.net"
}

// RequiresLogin checks if this module requires a login to work
func (m *pixiv) RequiresLogin() (requiresLogin bool) {
	return true
}

// IsLoggedIn returns the logged in status
func (m *pixiv) IsLoggedIn() bool {
	return m.LoggedIn
}

// RegisterURISchema adds our pattern to the URI Schemas
func (m *pixiv) RegisterURISchema(uriSchemas map[string][]*regexp.Regexp) {
	uriSchemas[m.Key()] = []*regexp.Regexp{
		regexp.MustCompile(".*pixiv.(co.jp|net)"),
	}
}

// AddSettingsCommand adds custom module specific settings and commands to our application
func (m *pixiv) AddSettingsCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
}

// Login logs us in for the current session if possible/account available
func (m *pixiv) Login(account *models.Account) bool {
	data := url.Values{
		"device_token":   {"pixiv"},
		"get_secure_url": {"true"},
		"include_policy": {"true"},
		"client_id":      {m.pixivSession.API.ClientID},
		"client_secret":  {m.pixivSession.API.ClientSecret},
	}

	if m.pixivSession.API.RefreshToken != "" {
		data.Set("grant_type", "refresh_token")
		data.Set("refresh_token", m.pixivSession.API.RefreshToken)
	} else {
		data.Set("grant_type", "password")
		data.Set("username", account.Username)
		data.Set("password", account.Password)
	}

	res, err := m.Session.Post(m.pixivSession.API.OauthURL, data)
	raven.CheckError(err)

	body, err := ioutil.ReadAll(res.Body)
	raven.CheckError(err)

	var response loginResponse
	_ = json.Unmarshal(body, &response)

	// check if the response could be parsed properly and save tokens
	if response.Response != nil {
		m.LoggedIn = true
		m.TriedLogin = true
		m.pixivSession.API.RefreshToken = response.Response.RefreshToken
		m.pixivSession.API.AccessToken = response.Response.AccessToken
	} else {
		var response errorResponse
		_ = json.Unmarshal(body, &response)
		log.WithField("module", m.Key()).Warning("login not successful.")
		raven.CheckError(
			fmt.Errorf("message: %s (code: %s)",
				response.Errors.System.Message,
				response.Errors.System.Code.String(),
			),
		)
	}

	return m.LoggedIn
}

// Parse parses the tracked item
func (m *pixiv) Parse(item *models.TrackedItem) error {
	switch {
	case strings.Contains(item.URI, "illust_id=") || strings.Contains(item.URI, "artworks"):
		err := m.parseUserIllustration(item)
		if err == nil {
			m.DbIO.ChangeTrackedItemCompleteStatus(item, true)
		}

		return err
	case strings.Contains(item.URI, "/member.php") || strings.Contains(item.URI, "/member_illust.php"):
		return m.parseUserIllustrations(item)
	case strings.Contains(item.URI, "/search.php"):
		return m.parseSearch(item)
	}

	return nil
}
