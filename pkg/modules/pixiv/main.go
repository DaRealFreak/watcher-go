package pixiv

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"regexp"
	"strings"

	"github.com/DaRealFreak/watcher-go/pkg/animation"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/modules/pixiv/session"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	log "github.com/sirupsen/logrus"
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
	SearchOrderAscending  = "asc"
	SearchOrderDescending = "desc"
)

// NewModule generates new module and registers the URI schema
func NewModule(dbIO models.DatabaseInterface, uriSchemas map[string][]*regexp.Regexp) *models.Module {
	// register empty sub module to point to
	var subModule = pixiv{
		animationHelper: animation.NewAnimationHelper(),
		pixivSession:    session.NewSession(),
	}

	// initialize the Module with the session/database and login status
	module := models.Module{
		DbIO:            dbIO,
		Session:         subModule.pixivSession,
		LoggedIn:        false,
		ModuleInterface: &subModule,
	}
	// set the module implementation for access to the session, database, etc
	subModule.Module = module
	subModule.pixivSession.Module = &subModule

	// register the uri schema
	module.RegisterURISchema(uriSchemas)
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
	var moduleURISchemas []*regexp.Regexp
	moduleURISchema := regexp.MustCompile(".*pixiv.(co.jp|net)")
	moduleURISchemas = append(moduleURISchemas, moduleURISchema)
	uriSchemas[m.Key()] = moduleURISchemas
}

// Login logs us in for the current session if possible/account available
func (m *pixiv) Login(account *models.Account) bool {
	data := url.Values{
		"get_secure_url": {"1"},
		"client_id":      {m.pixivSession.MobileClient.ClientID},
		"client_secret":  {m.pixivSession.MobileClient.ClientSecret},
	}

	if m.pixivSession.MobileClient.RefreshToken != "" {
		data.Set("grant_type", "refresh_token")
		data.Set("refresh_token", m.pixivSession.MobileClient.RefreshToken)
	} else {
		data.Set("grant_type", "password")
		data.Set("username", account.Username)
		data.Set("password", account.Password)
	}

	res, err := m.Session.Post(m.pixivSession.MobileClient.OauthURL, data)
	raven.CheckError(err)

	body, err := ioutil.ReadAll(res.Body)
	raven.CheckError(err)

	var response loginResponse
	_ = json.Unmarshal(body, &response)

	// check if the response could be parsed properly and save tokens
	if response.Response != nil {
		m.LoggedIn = true
		m.pixivSession.MobileClient.RefreshToken = response.Response.RefreshToken
		m.pixivSession.MobileClient.AccessToken = response.Response.AccessToken
	} else {
		var response errorResponse
		_ = json.Unmarshal(body, &response)
		log.Warning("login not successful.")
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
func (m *pixiv) Parse(item *models.TrackedItem) {
	if strings.Contains(item.URI, "/member.php") || strings.Contains(item.URI, "/member_illust.php") {
		m.parseUserIllustrations(item)
	}
}
