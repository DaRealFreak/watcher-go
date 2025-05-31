// Package deviantart contains the implementation of the deviantart module
package deviantart

import (
	"fmt"
	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/modules/deviantart/napi"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/PuerkitoBio/goquery"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/time/rate"
	"regexp"
	"sync"
	"time"
)

// deviantArt contains the implementation of the ModuleInterface
type deviantArt struct {
	*models.Module
	rateLimit  int
	nAPI       *napi.DeviantartNAPI
	daPattern  deviantArtPattern
	settings   deviantArtSettings
	proxies    []*proxySession
	multiProxy struct {
		currentIndexes []int
		waitGroup      sync.WaitGroup
	}
}

type deviantArtSettings struct {
	Loop        bool                 `mapstructure:"loop"`
	LoopProxies []http.ProxySettings `mapstructure:"loopproxies"`
	MultiProxy  bool                 `mapstructure:"multiproxy"`
	RateLimit   *int                 `mapstructure:"rate_limit"`
	Download    struct {
		DescriptionMinLength  int  `mapstructure:"description_min_length"`
		FollowForContent      bool `mapstructure:"follow_for_content"`
		UnfollowAfterDownload bool `mapstructure:"unfollow_after_download"`
	} `mapstructure:"download"`
	Cloudflare struct {
		UserAgent string `mapstructure:"user_agent"`
	} `mapstructure:"cloudflare"`
}

type deviantArtPattern struct {
	artPattern            *regexp.Regexp
	feedPattern           *regexp.Regexp
	userPattern           *regexp.Regexp
	galleryPattern        *regexp.Regexp
	scrapPattern          *regexp.Regexp
	collectionUUIDPattern *regexp.Regexp
	collectionPattern     *regexp.Regexp
	tagPattern            *regexp.Regexp
	searchPattern         *regexp.Regexp
}

// nolint: gochecknoinits
// init function registers the bare and the normal module to the module factories
func init() {
	modules.GetModuleFactory().RegisterModule(NewBareModule())
}

// NewBareModule returns a bare module implementation for the CLI options
func NewBareModule() *models.Module {
	module := &models.Module{
		Key:           "deviantart.com",
		RequiresLogin: true,
		LoggedIn:      false,
		URISchemas: []*regexp.Regexp{
			regexp.MustCompile(".*deviantart.com"),
			regexp.MustCompile(`deviantart://.*`),
		},
	}
	module.ModuleInterface = &deviantArt{
		Module:    module,
		daPattern: getDeviantArtPattern(),
	}

	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key,
		Color: "232:34",
	})

	return module
}

// InitializeModule initializes the module
func (m *deviantArt) InitializeModule() {
	// initialize settings
	raven.CheckError(viper.UnmarshalKey(
		fmt.Sprintf("Modules.%s", m.GetViperModuleKey()),
		&m.settings,
	))

	if m.settings.RateLimit != nil {
		m.rateLimit = *m.settings.RateLimit
	} else {
		// default rate limit of 1 request every 4 seconds
		m.rateLimit = 4000
	}
}

// AddModuleCommand adds custom module specific settings and commands to our application
func (m *deviantArt) AddModuleCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
}

// Login logs us in for the current session if possible/account available
func (m *deviantArt) Login(account *models.Account) bool {
	rateLimiter := rate.NewLimiter(rate.Every(time.Duration(m.rateLimit)*time.Millisecond), 1)
	m.nAPI = napi.NewDeviantartNAPI(m.Key, rateLimiter, m.settings.Cloudflare.UserAgent)
	// set the proxy if requested
	raven.CheckError(m.setProxyMethod())

	err := m.nAPI.Login(account)
	m.LoggedIn = err == nil

	// initialize proxy sessions after login
	if m.LoggedIn && m.settings.MultiProxy {
		m.initializeProxySessions()
		rateLimiter.SetLimit(rate.Every(time.Duration(m.rateLimit/len(m.proxies)) * time.Millisecond))
	}

	return m.LoggedIn
}

// extractCsrfToken extracts the CSRF token from the deviantArt website using the given item URI
func (m *deviantArt) extractCsrfToken(item *models.TrackedItem) (csrfToken string, err error) {
	res, requestErr := m.nAPI.UserSession.Get(item.URI)
	if requestErr != nil {
		return "", requestErr
	}

	jsonPattern := regexp.MustCompile(`.*\\"csrfToken\\":\\"(?P<Number>[^\\"]+)\\".*`)
	document, documentErr := goquery.NewDocumentFromReader(res.Body)
	if documentErr != nil {
		return "", err
	}

	scriptTags := document.Find("script")
	scriptTags.Each(func(row int, selection *goquery.Selection) {
		// no need for further checks if we already have the csrf token
		if csrfToken != "" {
			return
		}

		scriptContent := selection.Text()
		if jsonPattern.MatchString(scriptContent) {
			csrfToken = jsonPattern.FindStringSubmatch(scriptContent)[1]
		}
	})

	return csrfToken, nil
}

// Parse parses the tracked item
func (m *deviantArt) Parse(item *models.TrackedItem) (err error) {
	csrfToken, tokenErr := m.extractCsrfToken(item)
	if tokenErr != nil {
		return tokenErr
	}

	m.nAPI.CSRFToken = csrfToken
	log.WithField("module", m.Key).Debugf("extracted new CSRF token: %s", csrfToken)

	switch {
	case m.daPattern.artPattern.MatchString(item.URI):
		return m.parseArtNapi(item)
	case m.daPattern.feedPattern.MatchString(item.URI):
		return m.parseFeedNapi(item)
	case m.daPattern.userPattern.MatchString(item.URI):
		return m.parseUserNapi(item)
	case m.daPattern.galleryPattern.MatchString(item.URI):
		return m.parseGalleryNapi(item)
	case m.daPattern.scrapPattern.MatchString(item.URI):
		return m.parseGalleryNapi(item)
	case m.daPattern.collectionPattern.MatchString(item.URI):
		return m.parseCollectionNapi(item)
	case m.daPattern.collectionUUIDPattern.MatchString(item.URI):
		return m.parseCollectionUUIDNapi(item)
	case m.daPattern.tagPattern.MatchString(item.URI):
		return m.parseTagNapi(item)
	case m.daPattern.searchPattern.MatchString(item.URI):
		return m.parseSearchNapi(item)
	default:
		return fmt.Errorf("URL could not be associated with any of the implemented methods")
	}
}

// getDeviantArtPattern returns all required patterns
// extracted from the NewBareModule function to test in Unit Tests
func getDeviantArtPattern() deviantArtPattern {
	return deviantArtPattern{
		artPattern:            regexp.MustCompile(`https://www.deviantart.com/([^/?&]+?)/art/(?:[^/?&]+?)-(\d+)$`),
		userPattern:           regexp.MustCompile(`https://www.deviantart.com/([^/?&]+?)(?:/gallery|/gallery/all)?(?:/)?$`),
		feedPattern:           regexp.MustCompile(`https://www.deviantart.com(?:/)?$`),
		galleryPattern:        regexp.MustCompile(`https://www.deviantart.com/([^/?&]+?)/gallery/(\d+).*`),
		scrapPattern:          regexp.MustCompile(`https://www.deviantart.com/([^/?&]+?)/gallery/scraps`),
		collectionPattern:     regexp.MustCompile(`https://www.deviantart.com/([^/?&]+?)/favourites(?:/(\d+))?.*`),
		collectionUUIDPattern: regexp.MustCompile(`deviantart://collection/([^/?&]+?)/([^/?&]+)`),
		tagPattern:            regexp.MustCompile(`https://www.deviantart.com/tag/([^/?&]+)(?:$|/.*)`),
		searchPattern:         regexp.MustCompile(`https://www.deviantart.com/search.*q=(.*)`),
	}
}

// setProxyMethod determines what proxy method is being used and sets/updates the proxy configuration
func (m *deviantArt) setProxyMethod() error {
	switch {
	case m.settings.Loop && len(m.settings.LoopProxies) < 2:
		return fmt.Errorf("you need to at least register 2 proxies to loop")
	case !m.settings.Loop && m.GetProxySettings() != nil && m.GetProxySettings().Enable:
		return m.nAPI.UserSession.SetProxy(m.GetProxySettings())
	case m.settings.Loop:
		// reset proxy loop index if we reach the limit with the next iteration
		if m.ProxyLoopIndex+1 == len(m.settings.LoopProxies) {
			m.ProxyLoopIndex = -1
		}
		m.ProxyLoopIndex++

		return m.nAPI.UserSession.SetProxy(&m.settings.LoopProxies[m.ProxyLoopIndex])
	default:
		return nil
	}
}
