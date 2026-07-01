// Package deviantart contains the implementation of the deviantart module
package deviantart

import (
	"fmt"
	formatter "github.com/DaRealFreak/colored-nested-formatter/v2"
	"github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/modules/deviantart/browserlogin"
	"github.com/DaRealFreak/watcher-go/internal/modules/deviantart/napi"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/PuerkitoBio/goquery"
	fhttp "github.com/bogdanfinn/fhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/time/rate"
	"log/slog"
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
		SkipSourceDownloads   bool `mapstructure:"skip_source_downloads"`
	} `mapstructure:"download"`
	Cloudflare struct {
		UserAgent         string `mapstructure:"user_agent"`
		LoginWithoutProxy bool   `mapstructure:"login_without_proxy"`
	} `mapstructure:"cloudflare"`
	Login struct {
		// BrowserPath overrides the Chrome/Chromium executable used for the
		// PerimeterX-gated browser login. Empty auto-detects (and downloads
		// Chrome for Testing if none is installed).
		BrowserPath string `mapstructure:"browser_path"`
		// Headless controls whether the login browser runs without a window.
		// nil defaults to true; set to false to watch/debug the login.
		Headless *bool `mapstructure:"headless"`
	} `mapstructure:"login"`
	Debug struct {
		LogRequests bool   `mapstructure:"log_requests"`
		LogDir      string `mapstructure:"log_dir"`
	} `mapstructure:"debug"`
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
		SettingsSchema: deviantArtSettings{},
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
	var logger *napi.RequestLogger
	if m.settings.Debug.LogRequests {
		dir := m.settings.Debug.LogDir
		if dir == "" {
			dir = "da-requests"
		}
		logger = napi.NewRequestLogger(dir)
	}
	m.nAPI = napi.NewDeviantartNAPI(m.Key, m.settings.Cloudflare.UserAgent, logger)

	// set the proxy if requested
	if m.settings.Cloudflare.LoginWithoutProxy {
		slog.Debug("setting proxy to false for login without proxy", "module", m.Key)
		raven.CheckError(m.nAPI.UserSession.SetProxy(&http.ProxySettings{Enable: false}))
	}

	m.LoggedIn = m.authenticate(account)

	if m.settings.Cloudflare.LoginWithoutProxy {
		raven.CheckError(m.setProxyMethod())
	}

	// apply rate limiter to the session after login
	rateLimiter := rate.NewLimiter(rate.Every(time.Duration(m.rateLimit)*time.Millisecond), 1)
	m.nAPI.UserSession.SetRateLimiter(rateLimiter)

	// initialize proxy sessions after login
	if m.LoggedIn && m.settings.MultiProxy {
		m.initializeProxySessions()
		rateLimiter.SetLimit(rate.Every(time.Duration(m.rateLimit/len(m.proxies)) * time.Millisecond))
	}

	return m.LoggedIn
}

// authenticate establishes an authenticated session. DeviantArt's credential
// login is protected by PerimeterX bot detection, so it cannot be done with a
// plain HTTP client. The flow is:
//  1. reuse previously persisted session cookies if they still validate
//  2. otherwise perform a browser-based login and persist the harvested cookies
func (m *deviantArt) authenticate(account *models.Account) bool {
	// fast path: reuse a persisted session without launching a browser
	if m.loadStoredCookies() && m.nAPI.IsLoggedIn() {
		slog.Info("authenticated via stored session cookies", "module", m.Key)
		return true
	}

	if account == nil {
		slog.Error(
			"no valid session cookies and no account configured for the browser login",
			"module", m.Key,
		)
		return false
	}

	cookies, err := m.browserLogin(account)
	if err != nil {
		slog.Error(fmt.Sprintf("browser login failed: %v", err), "module", m.Key)
		return false
	}

	m.nAPI.SetSessionCookies(cookies)

	if !m.nAPI.IsLoggedIn() {
		slog.Error("browser login completed but the session did not validate", "module", m.Key)
		return false
	}

	m.persistCookies(cookies)
	slog.Info("authenticated via browser login", "module", m.Key)

	return true
}

// browserLogin performs the PerimeterX-gated login in a real browser. The Chrome
// executable is resolved by browserlogin (auto-detecting an installed browser, or
// downloading Chrome for Testing when the configured path is empty).
func (m *deviantArt) browserLogin(account *models.Account) ([]*fhttp.Cookie, error) {
	headless := true
	if m.settings.Login.Headless != nil {
		headless = *m.settings.Login.Headless
	}

	slog.Info("performing browser login for deviantart (PerimeterX)", "module", m.Key)

	return browserlogin.Login(account.Username, account.Password, browserlogin.Options{
		ChromePath: m.settings.Login.BrowserPath,
		Headless:   headless,
	})
}

// loadStoredCookies injects the module's persisted cookies into the session and
// reports whether any (enabled) cookies were loaded.
func (m *deviantArt) loadStoredCookies() bool {
	cookies := m.DbIO.GetAllCookies(m)

	sessionCookies := make([]*fhttp.Cookie, 0, len(cookies))
	for _, cookie := range cookies {
		if cookie.Disabled {
			continue
		}
		sessionCookies = append(sessionCookies, &fhttp.Cookie{
			Name:   cookie.Name,
			Value:  cookie.Value,
			Domain: ".deviantart.com",
			Path:   "/",
		})
	}

	if len(sessionCookies) == 0 {
		return false
	}

	m.nAPI.SetSessionCookies(sessionCookies)

	return true
}

// persistCookies upserts the harvested session cookies so later runs can reuse
// them without launching a browser (until they expire).
func (m *deviantArt) persistCookies(cookies []*fhttp.Cookie) {
	for _, cookie := range cookies {
		expiration := ""
		if !cookie.Expires.IsZero() {
			expiration = cookie.Expires.Format(time.RFC3339)
		}

		m.DbIO.GetFirstOrCreateCookie(cookie.Name, cookie.Value, expiration, m)
		m.DbIO.UpdateCookie(cookie.Name, cookie.Value, expiration, m)
	}
}

// extractCsrfToken extracts the CSRF token from the deviantArt website using the given item URI
func (m *deviantArt) extractCsrfToken(item *models.TrackedItem) (csrfToken string, err error) {
	res, requestErr := m.nAPI.UserSession.Get(item.URI)
	if requestErr != nil {
		return "", requestErr
	}
	defer func() { _ = res.Body.Close() }()

	jsonPattern := regexp.MustCompile(`.*window\.__CSRF_TOKEN__\s*=\s*'(?P<Number>[^']+)';.*`)
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
	slog.Debug(fmt.Sprintf("extracted new CSRF token: %s", csrfToken), "module", m.Key)

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

		for !m.settings.LoopProxies[m.ProxyLoopIndex].Enable {
			// skip to the next proxy if the current one is disabled
			m.ProxyLoopIndex++
			if m.ProxyLoopIndex+1 == len(m.settings.LoopProxies) {
				m.ProxyLoopIndex = -1
				break
			}
		}

		return m.nAPI.UserSession.SetProxy(&m.settings.LoopProxies[m.ProxyLoopIndex])
	default:
		return nil
	}
}
