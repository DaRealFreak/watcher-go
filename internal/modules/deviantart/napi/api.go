// Package napi is the implementation of the DeviantArt frontend API
package napi

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/DaRealFreak/watcher-go/internal/modules/deviantart/parser"
	http "github.com/bogdanfinn/fhttp"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	watcherHttp "github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/http/tls_session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/deviantart/login"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	"github.com/jaytaylor/html2text"
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
	AdditionalMedia []struct {
		FileId   json.Number `json:"fileId"`
		Type     string      `json:"type"`
		Width    json.Number `json:"width"`
		Height   json.Number `json:"height"`
		FileSize json.Number `json:"filesize"`
		Position json.Number `json:"position"`
		Media    *Media      `json:"media"`
	} `json:"additionalMedia"`
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
	Source   *string     `json:"s"`
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
	Account     *models.Account
	UserSession watcherHttp.TlsClientSessionInterface
	ctx         context.Context
	// FixMe: CSRF token is only valid for 30 minutes, we need to re-extract it after again
	CSRFToken string
	UserAgent string
	moduleKey string
}

// NewDeviantartNAPI returns the settings of the DeviantArt API
func NewDeviantartNAPI(moduleKey string, userAgent string) *DeviantartNAPI {
	return &DeviantartNAPI{
		UserSession: tls_session.NewTlsClientSession(moduleKey, DeviantArtErrorHandler{ModuleKey: moduleKey}),
		ctx:         context.Background(),
		moduleKey:   moduleKey,
		UserAgent:   userAgent,
	}
}

func (a *DeviantartNAPI) Login(account *models.Account) error {
	res, err := a.get("https://www.deviantart.com/users/login")
	if err != nil {
		return err
	}

	info, err := a.GetLoginCSRFToken(res)
	if err != nil {
		return err
	}

	if info.CSRFToken == "" {
		return fmt.Errorf("could not retrieve csrf_token from login page")
	}

	if info.LuToken == "" {
		return fmt.Errorf("could not retrieve lu_token token from login page")
	}

	a.CSRFToken = info.CSRFToken

	values := url.Values{
		"referer":      {"https://www.deviantart.com"},
		"referer_type": {""},
		"csrf_token":   {info.CSRFToken},
		"challenge":    {"0"},
		"lu_token":     {info.LuToken},
		"username":     {account.Username},
		"remember":     {"on"},
	}

	req, _ := http.NewRequest("POST", "https://www.deviantart.com/_sisu/do/step2", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err = a.do(req)
	if err != nil {
		return err
	}

	info, err = a.GetLoginCSRFToken(res)
	if err != nil {
		return err
	}

	if info.LuToken2 == "" {
		return fmt.Errorf("could not retrieve lu_token2 token from login page")
	}

	// update tokens, reset username (taken from lu_token2 from DA serverside) and add password
	values["csrf_token"] = []string{info.CSRFToken}
	values["lu_token"] = []string{info.LuToken}
	values["lu_token2"] = []string{info.LuToken2}
	values["username"] = []string{""}
	values["password"] = []string{account.Password}

	req, _ = http.NewRequest(
		"POST",
		"https://www.deviantart.com/_sisu/do/signin",
		strings.NewReader(values.Encode()),
	)

	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	req.Header.Set("Referer", "https://www.deviantart.com/_sisu/do/step2")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://www.deviantart.com")
	req.Header.Set("DNT", "1")
	req.Header.Set("Sec-GPC", "1")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Priority", "u=0, i")

	res, err = a.do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		return fmt.Errorf("login failed. status code: %d", res.StatusCode)
	}

	info, err = a.GetLoginCSRFToken(res)
	if err != nil {
		return err
	}

	// CSRFToken is required, LuToken and LuToken2 should be empty after a successful login
	if info.CSRFToken == "" || info.LuToken != "" || info.LuToken2 != "" {
		return fmt.Errorf("login failed")
	}

	a.CSRFToken = info.CSRFToken
	a.Account = account

	return nil
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

func (m *Media) GetPdfMedia() (bestMediaType *MediaType) {
	for _, mediaType := range m.Types {
		if mediaType.Types == "pdf" {
			return mediaType
		}
	}

	return nil
}

func (d *Draft) GetText() (text string) {
	for _, block := range d.Blocks {
		text += block.Text + "\n"
	}

	return text
}

func (d *TextContent) GetTextContent() (string, error) {
	switch d.Html.Type {
	case "draft":
		var draft Draft
		if err := json.Unmarshal([]byte(d.Html.Markup), &draft); err != nil {
			return "", err
		}
		return draft.GetText(), nil
	case "tiptap":
		html, err := parser.ParseTipTapFormat(d.Html.Markup)
		if err != nil {
			return "", err
		}
		return html2text.FromString(html)
	default:
		return html2text.FromString(d.Html.Markup)
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
