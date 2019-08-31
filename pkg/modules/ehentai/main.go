package ehentai

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/DaRealFreak/watcher-go/cmd/log/formatter"
	"github.com/DaRealFreak/watcher-go/pkg/http/session"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/time/rate"
)

// ehentai contains the implementation of the ModuleInterface and extends it by custom required values
type ehentai struct {
	models.Module
	downloadLimitReached     bool
	galleryImageIDPattern    *regexp.Regexp
	galleryImageIndexPattern *regexp.Regexp
	searchGalleryIDPattern   *regexp.Regexp
}

// NewModule generates new module and registers the URI schema
func NewModule(dbIO models.DatabaseInterface, uriSchemas map[string][]*regexp.Regexp) *models.Module {
	// register empty sub module to point to
	var subModule = ehentai{
		galleryImageIDPattern:    regexp.MustCompile(`(\w+-\d+)`),
		galleryImageIndexPattern: regexp.MustCompile(`\w+-(?P<Number>\d+)`),
		searchGalleryIDPattern:   regexp.MustCompile(`(\d+/\w+)`),
	}

	// set rate limiter on 1.5 seconds with burst limit of 1
	ehSession := session.NewSession()
	ehSession.RateLimiter = rate.NewLimiter(rate.Every(1500*time.Millisecond), 1)
	ehSession.ModuleKey = subModule.Key()

	// initialize the Module with the session/database and login status
	module := models.Module{
		DbIO:            dbIO,
		Session:         ehSession,
		LoggedIn:        false,
		ModuleInterface: &subModule,
	}
	// set the module implementation for access to the session, database, etc
	subModule.Module = module

	// register the uri schema
	module.RegisterURISchema(uriSchemas)
	return &module
}

// Key returns the module key
func (m *ehentai) Key() (key string) {
	return "e-hentai.com"
}

// RequiresLogin checks if this module requires a login to work
func (m *ehentai) RequiresLogin() (requiresLogin bool) {
	return false
}

// IsLoggedIn returns the logged in status
func (m *ehentai) IsLoggedIn() bool {
	return m.LoggedIn
}

// RegisterURISchema adds our pattern to the URI Schemas
func (m *ehentai) RegisterURISchema(uriSchemas map[string][]*regexp.Regexp) {
	var moduleURISchemas []*regexp.Regexp
	moduleURISchema := regexp.MustCompile(`.*e[\-x]hentai.org`)
	moduleURISchemas = append(moduleURISchemas, moduleURISchema)
	uriSchemas[m.Key()] = moduleURISchemas
}

// Login logs us in for the current session if possible/account available
func (m *ehentai) Login(account *models.Account) bool {
	values := url.Values{
		"CookieDate":       {"1"},
		"b":                {"d"},
		"bt":               {"1-1"},
		"UserName":         {account.Username},
		"PassWord":         {account.Password},
		"ipb_login_submit": {"Login!"},
	}

	res, _ := m.Session.Post("https://forums.e-hentai.org/index.php?act=Login&CODE=01", values)
	htmlResponse, _ := m.Session.GetDocument(res).Html()
	m.LoggedIn = strings.Contains(htmlResponse, "You are now logged in")

	// copy the cookies for e-hentai to exhentai
	ehURL, _ := url.Parse("https://e-hentai.org")
	exURL, _ := url.Parse("https://exhentai.org")
	m.Session.GetClient().Jar.SetCookies(exURL, m.Session.GetClient().Jar.Cookies(ehURL))

	return m.LoggedIn
}

// Parse parses the tracked item
func (m *ehentai) Parse(item *models.TrackedItem) {
	if strings.Contains(item.URI, "/g/") && !m.downloadLimitReached {
		m.parseGallery(item)
	} else if strings.Contains(item.URI, "/tag/") || strings.Contains(item.URI, "f_search=") {
		m.parseSearch(item)
	}
}

// processDownloadQueue processes the download queue consisting of gallery items
func (m *ehentai) processDownloadQueue(downloadQueue []imageGalleryItem, trackedItem *models.TrackedItem) {
	log.WithField("module", m.Key()).Info(
		fmt.Sprintf("found %d new items for uri: \"%s\"", len(downloadQueue), trackedItem.URI),
	)

	for index, data := range downloadQueue {
		downloadQueueItem := m.getDownloadQueueItem(data)
		// check for limit
		if downloadQueueItem.FileURI == "https://exhentai.org/img/509.gif" ||
			downloadQueueItem.FileURI == "https://e-hentai.org/img/509.gif" {
			log.WithField("module", m.Key()).Info("download limit reached, skipping galleries from now on")
			m.downloadLimitReached = true
			break
		}

		log.WithField("module", m.Key()).Info(
			fmt.Sprintf(
				"downloading updates for uri: \"%s\" (%0.2f%%)",
				trackedItem.URI,
				float64(index+1)/float64(len(downloadQueue))*100,
			),
		)
		raven.CheckError(
			m.Session.DownloadFile(
				path.Join(
					viper.GetString("download.directory"),
					m.Key(),
					downloadQueueItem.DownloadTag, downloadQueueItem.FileName,
				),
				downloadQueueItem.FileURI,
			),
		)
		// if no error occurred update the tracked item
		m.DbIO.UpdateTrackedItem(trackedItem, downloadQueueItem.ItemID)
	}
}

// init registers the module to the log formatter
func init() {
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: (&ehentai{}).Key(),
		Color: "232:94",
	})
}
