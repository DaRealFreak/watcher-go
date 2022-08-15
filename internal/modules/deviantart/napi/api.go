// Package napi is the implementation of the DeviantArt frontend API
package napi

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	cloudflarebp "github.com/DaRealFreak/cloudflare-bp-go"
	watcherHttp "github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/http/session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/deviantart/login"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"golang.org/x/time/rate"
)

const MaxLimit = 60

type Author struct {
	UserId     json.Number `json:"userId"`
	UseridUuid string      `json:"useridUuid"`
	Username   string      `json:"username"`
}

type Collection struct {
	FolderId       json.Number `json:"folderId"`
	CollectionUuid string      `json:"gallectionUuid"`
	Type           string      `json:"type"`
	Description    string      `json:"description"`
	Owner          *Author     `json:"owner"`
	Size           json.Number `json:"size"`
}

type Deviation struct {
	DeviationId    json.Number `json:"deviationId"`
	Type           string      `json:"type"`
	TypeID         json.Number `json:"typeId"`
	URL            string      `json:"url"`
	Title          string      `json:"title"`
	IsJournal      bool        `json:"isJournal"`
	IsVideo        bool        `json:"isVideo"`
	PublishedTime  string      `json:"publishedTime"`
	IsDeleted      bool        `json:"isDeleted"`
	IsDownloadable bool        `json:"isDownloadable"`
	IsBlocked      bool        `json:"isBlocked"`
	Author         *Author     `json:"author"`
	Media          *Media      `json:"media"`
}

type Media struct {
	BaseUri    string       `json:"baseUri"`
	PrettyName string       `json:"prettyName"`
	Types      []*MediaType `json:"types"`
}

type MediaType struct {
	Types    string       `json:"t"`
	Height   json.Number  `json:"h"`
	Width    json.Number  `json:"w"`
	Quality  string       `json:"q"`
	FileSize *json.Number `json:"f"`
	URL      *string      `json:"b"`
}

// DeviantartNAPI contains all required items to communicate with the API
type DeviantartNAPI struct {
	login.DeviantArtLogin
	UserSession watcherHttp.SessionInterface
	rateLimiter *rate.Limiter
	ctx         context.Context
	moduleKey   string
}

// NewDeviantartNAPI returns the settings of the DeviantArt API
func NewDeviantartNAPI(moduleKey string) *DeviantartNAPI {
	return &DeviantartNAPI{
		UserSession: session.NewSession(moduleKey),
		rateLimiter: rate.NewLimiter(rate.Every(2*time.Second), 1),
		ctx:         context.Background(),
		moduleKey:   moduleKey,
	}
}

func (a *DeviantartNAPI) Login(account *models.Account) error {
	res, err := a.UserSession.Get("https://www.deviantart.com/users/login")
	if err != nil {
		return err
	}

	info, err := a.GetLoginCSRFToken(res)
	if err != nil {
		return err
	}

	if !(info.CSRFToken != "") {
		return fmt.Errorf("could not retrieve CSRF token from login page")
	}

	values := url.Values{
		"referer":    {"https://www.deviantart.com"},
		"csrf_token": {info.CSRFToken},
		"challenge":  {"0"},
		"username":   {account.Username},
		"password":   {account.Password},
		"remember":   {"on"},
	}

	res, err = a.UserSession.GetClient().PostForm("https://www.deviantart.com/_sisu/do/signin", values)
	if err != nil {
		return err
	}

	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if !strings.Contains(string(content), "\"loggedIn\":true") &&
		!strings.Contains(string(content), "\\\"isLoggedIn\\\":true") {
		return fmt.Errorf("login failed")
	}

	return nil
}

// AddRoundTrippers adds the round trippers for CloudFlare, adds a custom user agent
// and implements the implicit OAuth2 authentication and sets the Token round tripper
func (a *DeviantartNAPI) AddRoundTrippers(userAgent string) {
	client := a.UserSession.GetClient()
	// apply CloudFlare bypass
	options := cloudflarebp.GetDefaultOptions()
	if userAgent != "" {
		options.Headers["User-Agent"] = userAgent
	}

	client.Transport = cloudflarebp.AddCloudFlareByPass(client.Transport, options)
	client.Transport = a.setDeviantArtHeaders(client.Transport)
}

// mapAPIResponse maps the API response into the passed APIResponse type
func (a *DeviantartNAPI) mapAPIResponse(res *http.Response, apiRes interface{}) (err error) {
	out, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	content := string(out)

	if res.StatusCode >= 400 {
		var apiErr Error

		if err = json.Unmarshal([]byte(content), &apiErr); err == nil {
			return apiErr
		}

		return fmt.Errorf(`unknown error response: "%s"`, content)
	}

	// unmarshal the request content into the response struct
	if err = json.Unmarshal([]byte(content), &apiRes); err != nil {
		return err
	}

	return nil
}

// applyRateLimit waits until the leaky bucket can pass another request again
func (a *DeviantartNAPI) applyRateLimit() {
	raven.CheckError(a.rateLimiter.Wait(a.ctx))
}

func (m *Media) GetHighestQualityVideoType() (bestMediaType *MediaType) {
	fileSize := 0
	for _, mediaType := range m.Types {
		if mediaType.Types != "video" {
			continue
		}

		typeFileSize, _ := strconv.ParseInt(mediaType.FileSize.String(), 10, 64)
		if int(typeFileSize) >= fileSize {
			bestMediaType = mediaType
		}
	}

	return bestMediaType
}
