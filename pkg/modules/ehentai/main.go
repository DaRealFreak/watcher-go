// Package ehentai contains the implementation of the e-hentai/exhentai module
package ehentai

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/pkg/http/session"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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
	ehSession := session.NewSession(subModule.getProxySettings())
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
	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key(),
		Color: "232:94",
	})

	return &module
}

// Key returns the module key
func (m *ehentai) Key() (key string) {
	return "e-hentai.org"
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
	uriSchemas[m.Key()] = []*regexp.Regexp{
		regexp.MustCompile(`.*e[\-x]hentai.org`),
	}
}

// AddSettingsCommand adds custom module specific settings and commands to our application
func (m *ehentai) AddSettingsCommand(command *cobra.Command) {
	m.addProxyCommands(command)
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
func (m *ehentai) Parse(item *models.TrackedItem) error {
	if strings.Contains(item.URI, "/g/") && !m.downloadLimitReached {
		return m.parseGallery(item)
	} else if strings.Contains(item.URI, "/tag/") || strings.Contains(item.URI, "f_search=") {
		return m.parseSearch(item)
	}

	return nil
}

// processDownloadQueue processes the download queue consisting of gallery items
func (m *ehentai) processDownloadQueue(downloadQueue []imageGalleryItem, trackedItem *models.TrackedItem) error {
	log.WithField("module", m.Key()).Info(
		fmt.Sprintf("found %d new items for uri: \"%s\"", len(downloadQueue), trackedItem.URI),
	)

	for index, data := range downloadQueue {
		downloadQueueItem, err := m.getDownloadQueueItem(data)
		if err != nil {
			return err
		}

		// check for limit
		if downloadQueueItem.FileURI == "https://exhentai.org/img/509.gif" ||
			downloadQueueItem.FileURI == "https://e-hentai.org/img/509.gif" {
			log.WithField("module", m.Key()).Info("download limit reached, skipping galleries from now on")
			m.downloadLimitReached = true

			return fmt.Errorf("download limit reached")
		}

		log.WithField("module", m.Key()).Info(
			fmt.Sprintf(
				"downloading updates for uri: \"%s\" (%0.2f%%)",
				trackedItem.URI,
				float64(index+1)/float64(len(downloadQueue))*100,
			),
		)

		err = m.Session.DownloadFile(
			path.Join(
				viper.GetString("download.directory"),
				m.Key(),
				strings.TrimSpace(downloadQueueItem.DownloadTag),
				strings.TrimSpace(downloadQueueItem.FileName),
			),
			downloadQueueItem.FileURI,
		)
		if err != nil {
			return err
		}
		// if no error occurred update the tracked item
		m.DbIO.UpdateTrackedItem(trackedItem, downloadQueueItem.ItemID)
	}

	return nil
}
