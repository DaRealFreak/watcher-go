// Package ehentai contains the implementation of the e-hentai/exhentai module
package ehentai

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/http/session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/time/rate"
)

// ehentai contains the implementation of the ModuleInterface and extends it by custom required values
type ehentai struct {
	*models.Module
	downloadLimitReached     bool
	galleryImageIDPattern    *regexp.Regexp
	galleryImageIndexPattern *regexp.Regexp
	searchGalleryIDPattern   *regexp.Regexp
	settings                 *ehentaiSettings
	ehSession                *session.DefaultSession
	proxies                  []*proxySession
	multiProxy               struct {
		currentIndexes []int
		mutex          sync.Mutex
		waitGroup      sync.WaitGroup
	}
}

type ehentaiSettings struct {
	Loop        bool                 `mapstructure:"loop"`
	LoopProxies []http.ProxySettings `mapstructure:"loopproxies"`
	MultiProxy  bool                 `mapstructure:"multiproxy"`
	Search      struct {
		BlacklistedTags  []string `mapstructure:"blacklisted_tags"`
		CategorizeSearch bool     `mapstructure:"categorize_search"`
		InheritSubFolder bool     `mapstructure:"inherit_sub_folder"`
	} `mapstructure:"search"`
}

// nolint: gochecknoinits
// init function registers the bare and the normal module to the module factories
func init() {
	modules.GetModuleFactory().RegisterModule(NewBareModule())
}

// NewBareModule returns a bare module implementation for the CLI options
func NewBareModule() *models.Module {
	module := &models.Module{
		Key:           "e-hentai.org",
		RequiresLogin: false,
		LoggedIn:      false,
		URISchemas: []*regexp.Regexp{
			regexp.MustCompile(`.*e[\-x]hentai.org`),
		},
		ProxyLoopIndex: -1,
	}
	module.ModuleInterface = &ehentai{
		Module:                   module,
		galleryImageIDPattern:    regexp.MustCompile(`(\w+-\d+)`),
		galleryImageIndexPattern: regexp.MustCompile(`\w+-(?P<Number>\d+)`),
		searchGalleryIDPattern:   regexp.MustCompile(`(\d+)/\w+`),
	}

	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key,
		Color: "232:94",
	})

	return module
}

// InitializeModule initializes the module
func (m *ehentai) InitializeModule() {
	// initialize settings
	raven.CheckError(viper.UnmarshalKey(
		fmt.Sprintf("Modules.%s", m.GetViperModuleKey()),
		&m.settings,
	))

	// set rate limiter on 2.5 seconds with burst limit of 1
	m.ehSession = session.NewSession(m.Key, ErrorHandler{}, session.DefaultErrorHandler{})
	m.ehSession.RateLimiter = rate.NewLimiter(rate.Every(2500*time.Millisecond), 1)
	m.Session = m.ehSession

	// set the proxy if requested
	raven.CheckError(m.setProxyMethod())
}

// AddModuleCommand adds custom module specific settings and commands to our application
func (m *ehentai) AddModuleCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
	m.AddProxyLoopCommands(command)
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

	// initialize proxy sessions after login
	if m.LoggedIn && m.settings.MultiProxy {
		m.initializeProxySessions()
	}

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
func (m *ehentai) processDownloadQueue(downloadQueue []*imageGalleryItem, trackedItem *models.TrackedItem) error {
	log.WithField("module", m.Key).Info(
		fmt.Sprintf("found %d new items for uri: \"%s\"", len(downloadQueue), trackedItem.URI),
	)

	for index, data := range downloadQueue {
		log.WithField("module", m.Key).Info(
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

func (m *ehentai) downloadItem(trackedItem *models.TrackedItem, data *imageGalleryItem) error {
	downloadQueueItem, err := m.getDownloadQueueItem(m.Session, trackedItem, data)
	if err != nil {
		return err
	}

	if err = m.downloadImage(trackedItem, downloadQueueItem); err != nil {
		if downloadQueueItem.FallbackFileURI != "" {
			data.uri = downloadQueueItem.FallbackFileURI
			fallback, fallbackErr := m.getDownloadQueueItem(m.Session, trackedItem, data)
			if fallbackErr != nil {
				return fallbackErr
			}

			log.WithField("module", m.Key).Warnf(
				"received status code 404 on gallery url \"%s\", trying fallback url \"%s\"",
				data.uri,
				fallback.FileURI,
			)

			downloadQueueItem.FileURI = fallback.FileURI
			downloadQueueItem.FallbackFileURI = ""

			// retry the fallback once and return that error (if occurred)
			return m.downloadImage(trackedItem, downloadQueueItem)
		}
		// if not returned from the previous checks just return the error
		return err
	}

	// if no error occurred update the tracked item
	m.DbIO.UpdateTrackedItem(trackedItem, downloadQueueItem.ItemID)

	return nil
}

func (m *ehentai) downloadImage(trackedItem *models.TrackedItem, downloadQueueItem *models.DownloadQueueItem) error {
	// check for limit
	if downloadQueueItem.FileURI == "https://exhentai.org/img/509.gif" ||
		downloadQueueItem.FileURI == "https://e-hentai.org/img/509.gif" {
		if m.settings.Loop {
			if err := m.setProxyMethod(); err != nil {
				return err
			}

			return m.downloadImage(trackedItem, downloadQueueItem)
		}

		log.WithField("module", m.Key).Info("download limit reached, skipping galleries from now on")
		m.downloadLimitReached = true

		return fmt.Errorf("download limit reached")
	}

	return m.Session.DownloadFile(
		path.Join(
			viper.GetString("download.directory"),
			m.Key,
			fp.TruncateMaxLength(fp.SanitizePath(strings.TrimSpace(trackedItem.SubFolder), false)),
			fp.TruncateMaxLength(strings.TrimSpace(downloadQueueItem.DownloadTag)),
			fp.TruncateMaxLength(strings.TrimSpace(downloadQueueItem.FileName)),
		),
		downloadQueueItem.FileURI,
	)
}

// setProxyMethod determines what proxy method is being used and sets/updates the proxy configuration
func (m *ehentai) setProxyMethod() error {
	switch {
	case m.settings == nil:
		return nil
	case m.settings.Loop && len(m.settings.LoopProxies) < 2:
		return fmt.Errorf("you need to at least register 2 proxies to loop")
	case !m.settings.Loop && m.GetProxySettings() != nil && m.GetProxySettings().Enable:
		return m.ehSession.SetProxy(m.GetProxySettings())
	case m.settings.Loop:
		if m.settings.MultiProxy {
			// ToDo:
			// function to check for free proxy
			// return new session (copied cookies from main session for login) with proxy applied
			// free proxy after download
			return m.ehSession.SetProxy(&m.settings.LoopProxies[0])
		} else {
			// reset proxy loop index if we reach the limit with the next iteration
			if m.ProxyLoopIndex+1 == len(m.settings.LoopProxies) {
				m.ProxyLoopIndex = -1
			}
			m.ProxyLoopIndex++

			return m.ehSession.SetProxy(&m.settings.LoopProxies[m.ProxyLoopIndex])
		}
	default:
		return nil
	}
}
