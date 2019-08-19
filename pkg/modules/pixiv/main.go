package pixiv

import (
	"encoding/json"
	"github.com/DaRealFreak/watcher-go/pkg/animation"
	"github.com/DaRealFreak/watcher-go/pkg/database"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/modules/pixiv/session"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/url"
	"regexp"
	"strings"
)

type pixiv struct {
	models.Module
	pixivSession    *session.PixivSession
	animationHelper *animation.Helper
}

type downloadQueueItem struct {
	ItemId       string
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

// generate new module and register uri schema
func NewModule(dbIO *database.DbIO, uriSchemas map[string][]*regexp.Regexp) *models.Module {
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
	module.RegisterUriSchema(uriSchemas)
	return &module
}

// retrieve the module key
func (m *pixiv) Key() (key string) {
	return "pixiv.net"
}

// check if this module requires a login to work
func (m *pixiv) RequiresLogin() (requiresLogin bool) {
	return true
}

// retrieve the logged in status
func (m *pixiv) IsLoggedIn() (LoggedIn bool) {
	return m.LoggedIn
}

// add our pattern to the uri schemas
func (m *pixiv) RegisterUriSchema(uriSchemas map[string][]*regexp.Regexp) {
	var moduleUriSchemas []*regexp.Regexp
	test, _ := regexp.Compile(".*pixiv.(co.jp|net)")
	moduleUriSchemas = append(moduleUriSchemas, test)
	uriSchemas[m.Key()] = moduleUriSchemas
}

// login function
func (m *pixiv) Login(account *models.Account) bool {
	data := url.Values{
		"get_secure_url": {"1"},
		"client_id":      {m.pixivSession.MobileClient.ClientId},
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

	res, err := m.Session.Post(m.pixivSession.MobileClient.OauthUrl, data)
	m.CheckError(err)

	body, err := ioutil.ReadAll(res.Body)
	m.CheckError(err)

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
		log.Fatalf("message: %s (code: %s)",
			response.Errors.System.Message,
			response.Errors.System.Code.String(),
		)
	}
	return m.LoggedIn
}

func (m *pixiv) Parse(item *models.TrackedItem) {
	if strings.Contains(item.Uri, "/member.php") || strings.Contains(item.Uri, "/member_illust.php") {
		m.parseUserIllustrations(item)
	}
}
