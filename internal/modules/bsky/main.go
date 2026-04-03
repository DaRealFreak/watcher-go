// Package bsky contains the implementation of the bsky.app module
package bsky

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	formatter "github.com/DaRealFreak/colored-nested-formatter/v2"
	"github.com/DaRealFreak/watcher-go/internal/http/tls_session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// bsky contains the implementation of the ModuleInterface
type bsky struct {
	*models.Module
	settings     bskySettings
	accessToken  string
	refreshToken string
	authPDS      string
}

type bskySettings struct {
	PDS string `mapstructure:"pds"`
}

// nolint: gochecknoinits
// init function registers the bare module to the module factory
func init() {
	modules.GetModuleFactory().RegisterModule(NewBareModule())
}

// NewBareModule returns a bare module implementation for the CLI options
func NewBareModule() *models.Module {
	module := &models.Module{
		Key:           "bsky.app",
		RequiresLogin: false,
		LoggedIn:      false,
		URISchemas: []*regexp.Regexp{
			regexp.MustCompile(`.*bsky\.app`),
		},
		SettingsSchema: bskySettings{},
	}
	module.ModuleInterface = &bsky{
		Module: module,
	}

	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key,
		Color: "255:33",
	})

	return module
}

// InitializeModule initializes the module
func (m *bsky) InitializeModule() {
	raven.CheckError(viper.UnmarshalKey(
		fmt.Sprintf("Modules.%s", m.GetViperModuleKey()),
		&m.settings,
	))

	if m.settings.PDS == "" {
		m.settings.PDS = "https://bsky.social"
	}

	m.Session = tls_session.NewTlsClientSession(m.Key)
	raven.CheckError(m.Session.SetProxy(m.GetProxySettings()))
}

// AddModuleCommand adds custom module specific settings and commands to our application
func (m *bsky) AddModuleCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
}

// SetCookies loads stored cookies and JWT tokens from the database
func (m *bsky) SetCookies() {
	m.Module.SetCookies()

	// load stored JWT tokens
	if cookie := m.DbIO.GetCookie("access_jwt", m); cookie != nil {
		m.accessToken = cookie.Value
	}
	if cookie := m.DbIO.GetCookie("refresh_jwt", m); cookie != nil {
		m.refreshToken = cookie.Value
	}
	if cookie := m.DbIO.GetCookie("auth_pds", m); cookie != nil {
		m.authPDS = cookie.Value
	}
}

// Login logs us in for the current session if possible/account available
func (m *bsky) Login(account *models.Account) bool {
	// if we have a valid access token from cookies, use it
	if m.accessToken != "" && m.authPDS != "" {
		m.LoggedIn = true
		return true
	}

	// try refreshing if we have a refresh token
	if m.refreshToken != "" && m.authPDS != "" {
		if err := m.doRefreshSession(); err == nil {
			m.LoggedIn = true
			return true
		}
	}

	// fresh login with credentials
	if err := m.createAuthSession(account.Username, account.Password); err != nil {
		slog.Error(
			fmt.Sprintf("failed to login to bsky: %s", err.Error()),
			"module", m.Key,
		)
		return false
	}

	m.LoggedIn = true
	return true
}

// Parse parses the tracked item
func (m *bsky) Parse(item *models.TrackedItem) error {
	return m.parseProfile(item)
}

// AddItem normalizes the URI before adding it to the database
func (m *bsky) AddItem(uri string) (string, error) {
	return strings.TrimRight(uri, "/"), nil
}
