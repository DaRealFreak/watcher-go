// Package tapas contains the implementation of the tapas.io module.
package tapas

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	formatter "github.com/DaRealFreak/colored-nested-formatter/v2"
	"github.com/DaRealFreak/watcher-go/internal/http/tls_session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/modules/tapas/api"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/spf13/cobra"
	"golang.org/x/time/rate"
)

type tapas struct {
	*models.Module
	api            *api.Client
	seriesURI      *regexp.Regexp
	episodeURI     *regexp.Regexp
	seriesIDURI    *regexp.Regexp
	seriesSlugURI  *regexp.Regexp
}

// nolint: gochecknoinits
func init() {
	modules.GetModuleFactory().RegisterModule(NewBareModule())
}

// NewBareModule returns the bare module instance used for CLI registration.
func NewBareModule() *models.Module {
	module := &models.Module{
		Key:           "tapas.io",
		RequiresLogin: false,
		LoggedIn:      false,
		URISchemas: []*regexp.Regexp{
			regexp.MustCompile(`tapas\.io`),
		},
	}
	module.ModuleInterface = &tapas{
		Module:        module,
		seriesURI:     regexp.MustCompile(`tapas\.io/series/`),
		episodeURI:    regexp.MustCompile(`tapas\.io/episode/(\d+)`),
		seriesIDURI:   regexp.MustCompile(`tapas\.io/series/(\d+)(?:/|$)`),
		seriesSlugURI: regexp.MustCompile(`tapas\.io/series/([^/?#]+)`),
	}

	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key,
		Color: "232:75",
	})

	return module
}

// InitializeModule initializes the underlying HTTP session and API client.
func (m *tapas) InitializeModule() {
	session := tls_session.NewTlsClientSession(m.Key)
	session.RateLimiter = rate.NewLimiter(rate.Every(1500*time.Millisecond), 1)
	m.Session = session

	raven.CheckError(m.Session.SetProxy(m.GetProxySettings()))

	m.api = api.NewClient(m.Session)
}

// AddModuleCommand registers module specific CLI commands.
func (m *tapas) AddModuleCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
}

// Login is a no-op for the public tapas.io content.
func (m *tapas) Login(_ *models.Account) bool {
	m.TriedLogin = true
	return true
}

// AddItem normalizes the incoming URI before it is persisted as a tracked
// item. Series URLs are rewritten to their canonical numeric form so that the
// same series can only be tracked once regardless of the slug variant the
// user pastes.
func (m *tapas) AddItem(uri string) (string, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return uri, err
	}

	parsed.Scheme = "https"
	parsed.Host = "tapas.io"
	parsed.RawQuery = ""
	parsed.Fragment = ""

	switch {
	case m.episodeURI.MatchString(uri):
		// /episode/{id} — strip trailing slashes, keep as-is.
		parsed.Path = strings.TrimRight(parsed.Path, "/")
		return parsed.String(), nil
	case m.seriesURI.MatchString(uri):
		if m.Session == nil || m.api == nil {
			m.InitializeModule()
		}

		seriesID, err := m.resolveSeriesID(uri)
		if err != nil {
			return uri, err
		}

		parsed.Path = fmt.Sprintf("/series/%s", seriesID)
		return parsed.String(), nil
	}

	return uri, nil
}

// Parse routes the tracked item to the appropriate handler.
func (m *tapas) Parse(item *models.TrackedItem) error {
	if m.api == nil {
		m.InitializeModule()
	}

	if m.episodeURI.MatchString(item.URI) {
		return m.parseEpisodeItem(item)
	}

	return m.parseSeries(item)
}

// resolveSeriesID returns the numeric series id for any series URI, resolving
// the slug via the API when necessary.
func (m *tapas) resolveSeriesID(uri string) (string, error) {
	if match := m.seriesIDURI.FindStringSubmatch(uri); len(match) > 1 {
		return match[1], nil
	}

	match := m.seriesSlugURI.FindStringSubmatch(uri)
	if len(match) < 2 {
		return "", fmt.Errorf("could not extract series identifier from %q", uri)
	}

	slug := strings.TrimSuffix(match[1], "/info")
	return m.api.SeriesIDFromSlug(slug)
}
