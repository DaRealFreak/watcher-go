// Package napi is the implementation of the DeviantArt frontend API
package napi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	"github.com/jaytaylor/html2text"
	"golang.org/x/time/rate"
)

// DateLayout is the date layout for parsing json times from the API
const DateLayout = "2006-01-02T15:04:05-0700"

// MaxLimit is the max limit of results for API endpoints who accept a limit parameter
const MaxLimit = 60

// OrderMostRecent is the value for the order parameter to sort by most recent results
// the default API response would sort by recommendation (which is useless)
const OrderMostRecent = "most-recent"

// MediaTypeFullView is the best type by default if the original file is not accessible
const MediaTypeFullView = "fullview"

// ModuleNameFolders is the string value for the module listing all available folders (galleries and collections)
const ModuleNameFolders = "folders"

// FolderIdAllFolder is the int value for the folder ID of the "All" folder, which is not an official folder
const FolderIdAllFolder = -1

// PremiumFolderDataWatcherType is the string value for premium folders requiring you to watch the author
const PremiumFolderDataWatcherType = "watchers"

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
	Name           string      `json:"name"`
	Owner          *Author     `json:"owner"`
	Size           json.Number `json:"size"`
}

type Deviation struct {
	DeviationId       json.Number        `json:"deviationId"`
	Type              string             `json:"type"`
	TypeID            json.Number        `json:"typeId"`
	URL               string             `json:"url"`
	Title             string             `json:"title"`
	IsJournal         bool               `json:"isJournal"`
	IsVideo           bool               `json:"isVideo"`
	PublishedTime     string             `json:"publishedTime"`
	IsDeleted         bool               `json:"isDeleted"`
	IsDownloadable    bool               `json:"isDownloadable"`
	IsBlocked         bool               `json:"isBlocked"`
	Author            *Author            `json:"author"`
	Media             *Media             `json:"media"`
	TextContent       *TextContent       `json:"textContent"`
	Extended          *Extended          `json:"extended"`
	PremiumFolderData *PremiumFolderData `json:"premiumFolderData"`
}

type PremiumFolderData struct {
	Type      string `json:"type"`
	HasAccess bool   `json:"hasAccess"`
}

type TextContent struct {
	Excerpt string `json:"excerpt"`
	Html    struct {
		Type   string `json:"type"`
		Markup string `json:"markup"`
	} `json:"html"`
}

type Draft struct {
	Blocks []struct {
		Key               string        `json:"key"`
		Text              string        `json:"text"`
		Type              string        `json:"type"`
		Depth             json.Number   `json:"depth"`
		InlineStyleRanges []interface{} `json:"inlineStyleRanges"`
		EntityRanges      []interface{} `json:"entityRanges"`
		Data              interface{}   `json:"data"`
	} `json:"blocks"`
	EntityMap interface{} `json:"entityMap"`
}

type Extended struct {
	DeviationUuid string `json:"deviationUuid"`
	OriginalFile  *struct {
		Type     string      `json:"type"`
		Width    json.Number `json:"width"`
		Height   json.Number `json:"height"`
		Filesize json.Number `json:"filesize"`
	} `json:"originalFile"`
	Download *struct {
		URL      string      `json:"url"`
		Type     string      `json:"type"`
		Width    json.Number `json:"width"`
		Height   json.Number `json:"height"`
		FileSize json.Number `json:"filesize"`
	} `json:"download"`
	DescriptionText *TextContent `json:"descriptionText"`
}

type Media struct {
	BaseUri    string       `json:"baseUri"`
	PrettyName string       `json:"prettyName"`
	Types      []*MediaType `json:"types"`
	Token      *Token       `json:"token"`
}

type Token []string

type MediaType struct {
	Types    string      `json:"t"`
	Height   json.Number `json:"h"`
	Width    json.Number `json:"w"`
	Crop     string      `json:"c"`
	Quality  *string     `json:"q"`
	FileSize json.Number `json:"f"`
	URL      *string     `json:"b"`
}

type Folder struct {
	FolderId       json.Number `json:"folderId"`
	GallectionUuid string      `json:"gallectionUuid"`
	Type           string      `json:"type"`
	Name           string      `json:"name"`
	Owner          *Author     `json:"owner"`
}

type Overview struct {
	Gruser struct {
		Page struct {
			Modules []*struct {
				Name       string `json:"name"`
				ModuleData struct {
					DataKey string `json:"dataKey"`
					Folders struct {
						HasMore    bool          `json:"hasMore"`
						NextOffset json.Number   `json:"nextOffset"`
						Results    []*Collection `json:"results"`
					} `json:"folders"`
				} `json:"moduleData"`
			} `json:"modules"`
		} `json:"page"`
	} `json:"gruser"`
}

// DeviantartNAPI contains all required items to communicate with the API
type DeviantartNAPI struct {
	login.DeviantArtLogin
	account     *models.Account
	UserSession watcherHttp.SessionInterface
	rateLimiter *rate.Limiter
	ctx         context.Context
	csrfToken   string
	moduleKey   string
}

// NewDeviantartNAPI returns the settings of the DeviantArt API
func NewDeviantartNAPI(moduleKey string, rateLimiter *rate.Limiter) *DeviantartNAPI {
	userSession := session.NewSession(moduleKey, DeviantArtErrorHandler{ModuleKey: moduleKey})
	userSession.RateLimiter = rateLimiter

	return &DeviantartNAPI{
		UserSession: userSession,
		rateLimiter: rate.NewLimiter(rate.Every(4*time.Second), 1),
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

	a.csrfToken = info.CSRFToken

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

	info, err = a.GetLoginCSRFToken(res)
	if err != nil {
		return err
	}

	content, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if info.CSRFToken != "" &&
		!strings.Contains(string(content), "\"loggedIn\":true") &&
		!strings.Contains(string(content), "\\\"isLoggedIn\\\":true") {
		return fmt.Errorf("login failed")
	}

	a.csrfToken = info.CSRFToken
	a.account = account

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
func (a *DeviantartNAPI) mapAPIResponse(res *http.Response, apiRes interface{}) error {
	out, err := io.ReadAll(res.Body)
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

func (a *Author) GetUsernameUrl() string {
	return strings.ToLower(url.PathEscape(a.Username))
}

func (t *MediaType) GetCrop(prettyName string) string {
	return strings.ReplaceAll(t.Crop, "<prettyName>", prettyName)
}

func (d *Deviation) GetPrettyName() string {
	if d.Media.PrettyName != "" {
		return d.Media.PrettyName
	} else {
		return fmt.Sprintf(
			"%s_by_%s",
			strings.ToLower(strings.ReplaceAll(fp.SanitizePath(d.Title, false), " ", "_")),
			strings.ToLower(strings.ReplaceAll(fp.SanitizePath(d.Author.Username, false), " ", "_")),
		)
	}
}

func (m *Media) GetType(mediaTypeTitle string) *MediaType {
	for _, mediaType := range m.Types {
		if mediaType.Types == mediaTypeTitle {
			return mediaType
		}
	}
	return nil
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

func (d *Draft) GetText() (text string) {
	for _, block := range d.Blocks {
		text += block.Text + "\n"
	}

	return text
}

func (d *Deviation) GetLiteratureContent() (string, error) {
	if d.TextContent.Html.Type == "draft" {
		var draft Draft
		if err := json.Unmarshal([]byte(d.TextContent.Html.Markup), &draft); err != nil {
			return "", err
		}
		return draft.GetText(), nil
	} else {
		return html2text.FromString(d.TextContent.Html.Markup)
	}
}

func (d *Deviation) GetPublishedTimestamp() string {
	t, _ := time.Parse(DateLayout, d.PublishedTime)
	return strconv.Itoa(int(t.Unix()))
}

func (d *Deviation) GetPublishedTime() time.Time {
	t, _ := time.Parse(DateLayout, d.PublishedTime)
	return t
}

func (t *Token) GetToken() string {
	for _, singleToken := range *t {
		return singleToken
	}

	return ""
}

func (c *CollectionsUserResponse) FindFolderByFolderId(folderId int) *Collection {
	// all folder has always the folder id -1 while it has no id in the URL, so we set it here manually
	if folderId == 0 {
		folderId = FolderIdAllFolder
	}

	for _, collection := range c.Collections {
		if currentFolderId, err := collection.FolderId.Int64(); err == nil {
			if int(currentFolderId) == folderId {
				return collection
			}
		}
	}

	return nil
}

func (c *CollectionsUserResponse) FindFolderByFolderUuid(folderUuid string) *Collection {
	for _, collection := range c.Collections {
		if collection.CollectionUuid == folderUuid {
			return collection
		}
	}

	return nil
}

func (o *Overview) FindFolderByFolderId(folderId int) *Collection {
	for _, module := range o.Gruser.Page.Modules {
		if module.Name == "folders" {
			for _, folder := range module.ModuleData.Folders.Results {
				if singleFolderId, err := folder.FolderId.Int64(); err == nil && int(singleFolderId) == folderId {
					return folder
				}
			}
		}
	}

	return nil
}
