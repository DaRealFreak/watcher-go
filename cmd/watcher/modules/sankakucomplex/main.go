package sankakucomplex

import (
	"encoding/json"
	"fmt"
	"github.com/kubernetes/klog"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"
	"watcher-go/cmd/watcher/arguments"
	"watcher-go/cmd/watcher/database"
	"watcher-go/cmd/watcher/http_wrapper"
	"watcher-go/cmd/watcher/models"
)

type sankakuComplex struct {
	models.BaseModel
	dbCon    *database.DbIO
	session  *http_wrapper.Session
	loggedIn bool
}

type apiItem struct {
	Id               json.Number
	Rating           string
	Status           string
	Author           author
	SampleUrl        string `json:"sample_url"`
	SampleWidth      int    `json:"sample_width"`
	SampleHeight     int    `json:"sample_height"`
	PreviewUrl       string `json:"preview_url"`
	PreviewWidth     int    `json:"preview_width"`
	FileUrl          string `json:"file_url"`
	Width            int
	Height           int
	FileSize         int         `json:"file_size"`
	FileType         string      `json:"file_type"`
	CreatedAt        created     `json:"created_at"`
	HasChildren      bool        `json:"has_children"`
	HasComments      bool        `json:"has_comments"`
	HasNotes         bool        `json:"has_notes"`
	IsFavorite       bool        `json:"is_favorited"`
	UserVote         json.Number `json:"user_vote"`
	Md5              string
	ParentId         json.Number `json:"parent_id"`
	Change           int
	FavCount         json.Number `json:"fav_count"`
	RecommendedPosts json.Number `json:"recommended_posts"`
	RecommendedScore json.Number `json:"recommended_score"`
	VoteCount        json.Number `json:"vote_count"`
	TotalScore       json.Number `json:"total_score"`
	CommentCount     json.Number `json:"comment_count"`
	Source           string
	InVisiblePool    bool `json:"in_visible_pool"`
	IsPremium        bool `json:"is_premium"`
	Sequence         json.Number
	Tags             []tag
}

type author struct {
	Id           int
	Name         string
	Avatar       string
	AvatarRating string `json:"avatar_rating"`
}

type created struct {
	JsonClass string `json:"json_class"`
	S         int
	N         int
}

type tag struct {
	Id        int
	NameEn    string `json:"name_en"`
	NameJa    string `json:"name_ja"`
	Type      int
	Count     int
	PostCount int `json:"post_count"`
	PoolCount int `json:"pool_count"`
	Locale    string
	Rating    json.Number
	Name      string
}

// generate new module and register uri schema
func NewModule(dbIO *database.DbIO, uriSchemas map[string][]*regexp.Regexp) *models.Module {
	var subModule = sankakuComplex{
		dbCon:    dbIO,
		session:  http_wrapper.NewSession(),
		loggedIn: false,
	}

	module := models.Module{
		ModuleInterface: &subModule,
	}
	// register the uri schema
	module.RegisterUriSchema(uriSchemas)
	return &module
}

// retrieve the module key
func (m *sankakuComplex) Key() (key string) {
	return "chan.sankakucomplex.com"
}

// retrieve the logged in status
func (m *sankakuComplex) IsLoggedIn() (loggedIn bool) {
	return m.loggedIn
}

// add our pattern to the uri schemas
func (m *sankakuComplex) RegisterUriSchema(uriSchemas map[string][]*regexp.Regexp) {
	var moduleUriSchemas []*regexp.Regexp
	test, _ := regexp.Compile(".*.sankakucomplex.com")
	moduleUriSchemas = append(moduleUriSchemas, test)
	uriSchemas[m.Key()] = moduleUriSchemas
}

// login function
func (m *sankakuComplex) Login(account *models.Account) bool {
	values := url.Values{
		"url":            {""},
		"user[name]":     {account.Username},
		"user[password]": {account.Password},
		"commit":         {"Login"},
	}

	res, _ := m.post("https://chan.sankakucomplex.com/user/authenticate", values, 0)
	htmlResponse, _ := m.session.GetDocument(res).Html()
	m.loggedIn = strings.Contains(htmlResponse, "You are now logged in")
	return m.loggedIn
}

func (m *sankakuComplex) Parse(item *models.TrackedItem) {
	tag := m.extractItemTag(item)
	page := 0
	foundCurrentItem := false
	var downloadQueue []models.DownloadQueueItem

	for foundCurrentItem == false {
		page += 1
		apiUri := fmt.Sprintf("https://capi-v2.sankakucomplex.com/posts?lang=english&page=%d&limit=100&tags=%s", page, tag)
		response, _ := m.get(apiUri, 0)
		apiItems := m.parseApiResponse(response)
		for _, data := range apiItems {
			if string(data.Id) != item.CurrentItem {
				downloadQueue = append(downloadQueue, models.DownloadQueueItem{
					ItemId:      string(data.Id),
					DownloadTag: tag,
					FileName:    string(data.Id) + "_" + m.GetFileName(data.FileUrl),
					FileUri:     data.FileUrl,
				})
			} else {
				foundCurrentItem = true
				break
			}
		}

		// we reached the last possible page, break here
		if len(apiItems) != 100 {
			break
		}

	}

	m.processDownloadQueue(downloadQueue, item)
}

func (m *sankakuComplex) processDownloadQueue(downloadQueue []models.DownloadQueueItem, trackedItem *models.TrackedItem) {
	// reverse queue to get the oldest "new" item first and manually update it
	downloadQueue = m.ReverseDownloadQueueItems(downloadQueue)
	klog.Info(fmt.Sprintf("found %d new items for uri: \"%s\"", len(downloadQueue), trackedItem.Uri))

	for index, data := range downloadQueue {
		klog.Info(fmt.Sprintf("downloading updates for uri: \"%s\" (%0.2f%%)", trackedItem.Uri, float64(index+1)/float64(len(downloadQueue))*100))
		_ = m.session.DownloadFile(path.Join(*arguments.DownloadDirectory, m.Key(), data.DownloadTag, data.FileName), data.FileUri)
		m.dbCon.UpdateTrackedItem(trackedItem, data.ItemId)
	}
}

// parse the response from the API
func (m *sankakuComplex) parseApiResponse(response *http.Response) []apiItem {
	body, _ := ioutil.ReadAll(response.Body)
	var apiItems []apiItem
	_ = json.Unmarshal(body, &apiItems)
	return apiItems
}

// extract the tag from the passed item to use in the API request
func (m *sankakuComplex) extractItemTag(item *models.TrackedItem) string {
	u, _ := url.Parse(item.Uri)
	q, _ := url.ParseQuery(u.RawQuery)
	if len(q["tags"]) == 0 {
		log.Fatalf("parsed uri(%s) does not contain any \"tags\" tag", item.Uri)
	}
	return q["tags"][0]
}

// custom POST function to check for specific status codes and messages
func (m *sankakuComplex) post(uri string, data url.Values, tries int) (*http.Response, error) {
	res, err := m.session.Post(uri, data, tries)
	if err == nil && res.StatusCode == 429 {
		klog.Info(fmt.Sprintf("too many requests, sleeping '%d' seconds", tries+1*60))
		time.Sleep(time.Duration(tries+1*60) * time.Second)
		return m.post(uri, data, tries+1)
	}
	return res, err
}

// custom GET function to check for specific status codes and messages
func (m *sankakuComplex) get(uri string, tries int) (*http.Response, error) {
	res, err := m.session.Get(uri, tries)
	if err == nil && res.StatusCode == 429 {
		klog.Info(fmt.Sprintf("too many requests, sleeping '%d' seconds", tries+1*60))
		time.Sleep(time.Duration(tries+1*60) * time.Second)
		return m.get(uri, tries+1)
	}
	return res, err
}
