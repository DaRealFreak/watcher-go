package ehentai

import (
	"fmt"
	"github.com/DaRealFreak/watcher-go/pkg/database"
	"github.com/DaRealFreak/watcher-go/pkg/http/session"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"math/rand"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"
)

type timer struct {
	lastRequest time.Time
	minWaitTime time.Duration
	maxWaitTime time.Duration
}

type ehentai struct {
	models.Module
	downloadLimitReached     bool
	galleryImageIdPattern    *regexp.Regexp
	galleryImageIndexPattern *regexp.Regexp
	searchGalleryIdPattern   *regexp.Regexp
	timer                    *timer
}

// generate new module and register uri schema
func NewModule(dbIO *database.DbIO, uriSchemas map[string][]*regexp.Regexp) *models.Module {
	// register empty sub module to point to
	var subModule = ehentai{
		galleryImageIdPattern:    regexp.MustCompile("(\\w+-\\d+)"),
		galleryImageIndexPattern: regexp.MustCompile("\\w+-(?P<Number>\\d+)"),
		searchGalleryIdPattern:   regexp.MustCompile("(\\d+/\\w+)"),
		timer: &timer{
			lastRequest: time.Now().Add(-1.5 * 1000 * time.Millisecond),
			minWaitTime: 1.5 * 1000 * time.Millisecond,
			maxWaitTime: 2.5 * 1000 * time.Millisecond,
		},
	}

	// initialize the Module with the session/database and login status
	module := models.Module{
		DbIO:            dbIO,
		Session:         session.NewSession(),
		LoggedIn:        false,
		ModuleInterface: &subModule,
	}
	// set the module implementation for access to the session, database, etc
	subModule.Module = module
	// register the uri schema
	module.RegisterUriSchema(uriSchemas)
	return &module
}

// retrieve the module key
func (m *ehentai) Key() (key string) {
	return "e-hentai.com"
}

// check if this module requires a login to work
func (m *ehentai) RequiresLogin() (requiresLogin bool) {
	return false
}

// retrieve the logged in status
func (m *ehentai) IsLoggedIn() (LoggedIn bool) {
	return m.LoggedIn
}

// add our pattern to the uri schemas
func (m *ehentai) RegisterUriSchema(uriSchemas map[string][]*regexp.Regexp) {
	var moduleUriSchemas []*regexp.Regexp
	schema, _ := regexp.Compile(".*e[\\-x]hentai.org")
	moduleUriSchemas = append(moduleUriSchemas, schema)
	uriSchemas[m.Key()] = moduleUriSchemas
}

// login function
func (m *ehentai) Login(account *models.Account) bool {
	values := url.Values{
		"CookieDate":       {"1"},
		"b":                {"d"},
		"bt":               {"1-1"},
		"UserName":         {account.Username},
		"PassWord":         {account.Password},
		"ipb_login_submit": {"Login!"},
	}

	res, _ := m.post("https://forums.e-hentai.org/index.php?act=Login&CODE=01", values)
	htmlResponse, _ := m.Session.GetDocument(res).Html()
	m.LoggedIn = strings.Contains(htmlResponse, "You are now logged in")

	// copy the cookies for e-hentai to exhentai
	ehUrl, _ := url.Parse("https://e-hentai.org")
	exUrl, _ := url.Parse("https://exhentai.org")
	m.Session.GetClient().Jar.SetCookies(exUrl, m.Session.GetClient().Jar.Cookies(ehUrl))

	return m.LoggedIn
}

func (m *ehentai) Parse(item *models.TrackedItem) {
	if strings.Contains(item.Uri, "/g/") && m.downloadLimitReached == false {
		m.parseGallery(item)
	} else if strings.Contains(item.Uri, "/tag/") || strings.Contains(item.Uri, "f_search=") {
		m.parseSearch(item)
	}
}

// custom POST function to check for specific status codes and messages
func (m *ehentai) post(uri string, data url.Values) (*http.Response, error) {
	m.checkPassedDuration()
	res, err := m.Session.Post(uri, data)
	return res, err
}

// custom GET function to check for specific status codes and messages
func (m *ehentai) get(uri string) (*http.Response, error) {
	m.checkPassedDuration()
	res, err := m.Session.Get(uri)
	return res, err
}

// function to add a random delay before sending another request
// to prevent getting detected as harvesting software
func (m *ehentai) checkPassedDuration() {
	randomWaitTime := time.Duration(float64(m.timer.minWaitTime) + rand.Float64()*(float64(m.timer.maxWaitTime)-float64(m.timer.minWaitTime)))
	if !time.Now().After(time.Now().Add(randomWaitTime)) {
		sleepTime := time.Now().Add(randomWaitTime).Sub(time.Now())
		log.Debugf("sleeping for %s seconds", sleepTime.String())
		time.Sleep(sleepTime)
	}
}

func (m *ehentai) processDownloadQueue(downloadQueue []imageGalleryItem, trackedItem *models.TrackedItem) {
	log.Info(fmt.Sprintf("found %d new items for uri: \"%s\"", len(downloadQueue), trackedItem.Uri))

	for index, data := range downloadQueue {
		m.checkPassedDuration()

		downloadQueueItem := m.getDownloadQueueItem(data)
		// check for limit
		if downloadQueueItem.FileUri == "https://exhentai.org/img/509.gif" ||
			downloadQueueItem.FileUri == "https://e-hentai.org/img/509.gif" {
			log.Info("download limit reached, skipping galleries from now on")
			m.downloadLimitReached = true
			break
		}

		log.Info(fmt.Sprintf("downloading updates for uri: \"%s\" (%0.2f%%)", trackedItem.Uri, float64(index+1)/float64(len(downloadQueue))*100))
		err := m.Session.DownloadFile(path.Join(viper.GetString("downloadDirectory"), m.Key(), downloadQueueItem.DownloadTag, downloadQueueItem.FileName), downloadQueueItem.FileUri)
		m.CheckError(err)
		// if no error occurred update the tracked item
		m.DbIO.UpdateTrackedItem(trackedItem, downloadQueueItem.ItemId)
	}
}
