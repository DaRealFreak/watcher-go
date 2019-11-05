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
	"github.com/DaRealFreak/watcher-go/pkg/modules"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/time/rate"
)

// ehentai contains the implementation of the ModuleInterface and extends it by custom required values
type ehentai struct {
	*models.Module
	downloadLimitReached     bool
	ProxyLoopIndex           int
	galleryImageIDPattern    *regexp.Regexp
	galleryImageIndexPattern *regexp.Regexp
	searchGalleryIDPattern   *regexp.Regexp
	settings                 *ModuleConfiguration
	ehSession                *session.DefaultSession
}

// nolint: gochecknoinits
// init function registers the bare and the normal module to the module factories
func init() {
	bare := modules.GetModuleFactory(true)
	bare.RegisterModule(NewBareModule())

	factory := modules.GetModuleFactory(false)
	factory.RegisterModule(NewModule())
}

// NewBareModule returns a bare module implementation for the CLI options
func NewBareModule() *models.Module {
	return &models.Module{
		ModuleInterface: &ehentai{},
	}
}

// NewModule generates new module and registers the URI schema
func NewModule() *models.Module {
	// initialize module and set our module interface with our custom module
	module := &models.Module{
		LoggedIn: false,
	}
	subModule := &ehentai{
		Module:                   module,
		galleryImageIDPattern:    regexp.MustCompile(`(\w+-\d+)`),
		galleryImageIndexPattern: regexp.MustCompile(`\w+-(?P<Number>\d+)`),
		searchGalleryIDPattern:   regexp.MustCompile(`(\d+)/\w+`),
		ProxyLoopIndex:           -1,
	}
	module.ModuleInterface = subModule

	// initialize settings
	raven.CheckError(viper.UnmarshalKey(
		fmt.Sprintf("Modules.%s", subModule.GetViperModuleKey()),
		&subModule.settings,
	))

	// set rate limiter on 1.5 seconds with burst limit of 1
	subModule.ehSession = session.NewSession(subModule.Key())
	subModule.ehSession.RateLimiter = rate.NewLimiter(rate.Every(1500*time.Millisecond), 1)
	subModule.Session = subModule.ehSession

	// set the proxy if requested
	raven.CheckError(subModule.setProxyMethod())

	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: subModule.Key(),
		Color: "232:94",
	})

	return module
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
	m.AddProxyCommands(command)
	m.addProxyLoopCommands(command)
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
	m.TriedLogin = true

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
		log.WithField("module", m.Key()).Info(
			fmt.Sprintf(
				"downloading updates for uri: \"%s\" (%0.2f%%)",
				trackedItem.URI,
				float64(index+1)/float64(len(downloadQueue))*100,
			),
		)

		if err := m.downloadItem(trackedItem, data); err != nil {
			return err
		}
	}

	return nil
}

func (m *ehentai) downloadItem(trackedItem *models.TrackedItem, data imageGalleryItem) error {
	downloadQueueItem, err := m.getDownloadQueueItem(data)
	if err != nil {
		return err
	}

	// check for limit
	if downloadQueueItem.FileURI == "https://exhentai.org/img/509.gif" ||
		downloadQueueItem.FileURI == "https://e-hentai.org/img/509.gif" {
		if m.settings.Loop {
			if err := m.setProxyMethod(); err != nil {
				return err
			}

			return m.downloadItem(trackedItem, data)
		}

		log.WithField("module", m.Key()).Info("download limit reached, skipping galleries from now on")
		m.downloadLimitReached = true

		return fmt.Errorf("download limit reached")
	}

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

	return nil
}

// setProxyMethod determines what proxy method is being used and sets/updates the proxy configuration
func (m *ehentai) setProxyMethod() error {
	switch {
	case m.settings == nil:
		return nil
	case m.settings.Loop && len(m.settings.LoopProxies) < 2:
		return fmt.Errorf("you need to at least register 2 proxies to loop")
	case !m.settings.Loop && m.settings.Proxy.Enable:
		return m.ehSession.SetProxy(&m.settings.Proxy)
	case m.settings.Loop:
		// reset proxy loop index if we reach the limit with the next iteration
		if m.ProxyLoopIndex+1 == len(m.settings.LoopProxies) {
			m.ProxyLoopIndex = -1
		}
		m.ProxyLoopIndex++

		return m.ehSession.SetProxy(&m.settings.LoopProxies[m.ProxyLoopIndex])
	default:
		return nil
	}
}
