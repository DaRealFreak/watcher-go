// Package ehentai contains the implementation of the e-hentai/exhentai module
package ehentai

import (
	"fmt"
	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/http/std_session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/time/rate"
	"net/url"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ehentai contains the implementation of the ModuleInterface and extends it by custom-required values
type ehentai struct {
	*models.Module
	Session                  http.StdClientSessionInterface
	rateLimit                int
	downloadLimitReached     bool
	ipBanned                 bool
	galleryImageIDPattern    *regexp.Regexp
	galleryImageIndexPattern *regexp.Regexp
	searchGalleryIDPattern   *regexp.Regexp
	settings                 ehentaiSettings
	proxies                  []*proxySession
	multiProxy               struct {
		currentIndexes []int
		waitGroup      sync.WaitGroup
	}
}

type ehentaiSettings struct {
	Loop        bool                 `mapstructure:"loop"`
	LoopProxies []http.ProxySettings `mapstructure:"loopproxies"`
	MultiProxy  bool                 `mapstructure:"multiproxy"`
	Search      struct {
		LowPoweredTags    bool     `mapstructure:"low_powered_tags"`
		ExpungedGalleries bool     `mapstructure:"expunged_galleries"`
		BlacklistedTags   []string `mapstructure:"blacklisted_tags"`
		WhitelistedTags   []string `mapstructure:"whitelisted_tags"`
		CategorizeSearch  bool     `mapstructure:"categorize_search"`
		InheritSubFolder  bool     `mapstructure:"inherit_sub_folder"`
	} `mapstructure:"search"`
	RateLimit *int `mapstructure:"rate_limit"`
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

	if m.settings.RateLimit != nil {
		m.rateLimit = *m.settings.RateLimit
	} else {
		// default rate limit of 1 request every 2.5 seconds
		m.rateLimit = 2500
	}

	// set rate limiter on 2.5 seconds with burst limit of 1
	ehSession := std_session.NewStdClientSession(m.Key, ErrorHandler{}, std_session.StdClientErrorHandler{})
	ehSession.RateLimiter = rate.NewLimiter(rate.Every(time.Duration(m.rateLimit)*time.Millisecond), 1)

	m.Session = ehSession

	// set the proxy if requested
	raven.CheckError(m.setProxyMethod())

	// initialize the proxy sessions if multi-proxy is enabled
	if m.settings.MultiProxy {
		m.initializeProxySessions()
	}
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

	res, err := m.get("https://e-hentai.org/home.php")
	if err != nil {
		m.TriedLogin = true
		return false
	}

	// check if we have a current session based on the stored session cookies
	htmlResponse, _ := m.Session.GetDocument(res).Html()
	m.LoggedIn = strings.Contains(htmlResponse, "You are currently at")

	if !m.LoggedIn {
		// else try to log in (possibly broken due to them adding captchas sometimes)
		res, err = m.post("https://forums.e-hentai.org/index.php?act=Login&CODE=01", values)
		if err != nil {
			m.TriedLogin = true
			return false
		}

		htmlResponse, _ = m.Session.GetDocument(res).Html()
		m.LoggedIn = strings.Contains(htmlResponse, "You are now logged in") || strings.Contains(htmlResponse, "Logged in as:")
		m.TriedLogin = true
	}

	// copy the cookies for e-hentai to exhentai
	ehURL, _ := url.Parse("https://e-hentai.org")
	exURL, _ := url.Parse("https://exhentai.org")
	m.Session.SetCookies(exURL, m.Session.GetCookies(ehURL))

	// reinitialize proxy sessions after login to ensure they have the correct cookies
	if m.LoggedIn && m.settings.MultiProxy {
		m.initializeProxySessions()
	}

	return m.LoggedIn
}

// Parse parses the tracked item
func (m *ehentai) Parse(item *models.TrackedItem) (err error) {
	if m.ipBanned {
		return IpBanError{}
	}

	if strings.Contains(item.URI, "/g/") && !m.downloadLimitReached {
		err = m.parseGallery(item)
	} else if strings.Contains(item.URI, "/tag/") || strings.Contains(item.URI, "f_search=") {
		// change to next proxy to avoid IP ban
		err = m.setProxyMethod()
		if err != nil {
			return err
		}

		err = m.parseSearch(item)
	}

	if _, ok := err.(IpBanError); ok {
		m.ipBanned = true
	}

	if _, ok := err.(IpBanSearchError); ok {
		m.ipBanned = true
	}

	return err
}

// processDownloadQueue processes the download queue consisting of gallery items
func (m *ehentai) processDownloadQueue(downloadQueue []*imageGalleryItem, trackedItem *models.TrackedItem, notifications ...*models.Notification) error {
	log.WithField("module", m.Key).Info(
		fmt.Sprintf("found %d new items for uri: \"%s\"", len(downloadQueue), trackedItem.URI),
	)

	for _, notification := range notifications {
		log.WithField("module", m.Key).Log(
			notification.Level,
			notification.Message,
		)
	}

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
			m.GetDownloadDirectory(),
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
	case m.settings.Loop && len(m.settings.LoopProxies) < 2:
		return fmt.Errorf("you need to at least register 2 proxies to loop")
	case !m.settings.Loop && m.GetProxySettings() != nil && m.GetProxySettings().Enable:
		return m.Session.SetProxy(m.GetProxySettings())
	case m.settings.Loop:
		// reset proxy loop index if we reach the limit with the next iteration
		if m.ProxyLoopIndex+1 == len(m.settings.LoopProxies) {
			m.ProxyLoopIndex = -1
		}
		m.ProxyLoopIndex++

		for !m.settings.LoopProxies[m.ProxyLoopIndex].Enable {
			// skip to the next proxy if the current one is disabled
			m.ProxyLoopIndex++
			if m.ProxyLoopIndex+1 == len(m.settings.LoopProxies) {
				m.ProxyLoopIndex = -1
				break
			}
		}

		return m.Session.SetProxy(&m.settings.LoopProxies[m.ProxyLoopIndex])
	default:
		return nil
	}
}
