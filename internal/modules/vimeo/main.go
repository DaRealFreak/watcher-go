package vimeo

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"

	cloudflarebp "github.com/DaRealFreak/cloudflare-bp-go"
	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/internal/http/session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type vimeo struct {
	*models.Module
	defaultVideoURLPattern *regexp.Regexp
	masterJsonPattern      *regexp.Regexp
}

// init function registers the bare and the normal module to the module factories
func init() {
	modules.GetModuleFactory().RegisterModule(NewBareModule())
}

// NewBareModule returns a bare module implementation for the CLI options
func NewBareModule() *models.Module {
	module := &models.Module{
		Key:           "vimeo.com",
		RequiresLogin: false,
		LoggedIn:      false,
		URISchemas: []*regexp.Regexp{
			regexp.MustCompile(`vimeo.com`),
			regexp.MustCompile(`.*/master.json.*`),
		},
	}
	module.ModuleInterface = &vimeo{
		Module:                 module,
		defaultVideoURLPattern: regexp.MustCompile(`https://(?:player.)?vimeo.com/(?:video/)?(\d+)(?:$|/(\w+)$|\?.*h=(\w+)|\?.*)`),
		masterJsonPattern:      regexp.MustCompile(`.*/master.json.*`),
	}

	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key,
		Color: "231:39",
	})

	return module
}

// InitializeModule initializes the module
func (m *vimeo) InitializeModule() {
	// set the module implementation for access to the session, database, etc
	m.Session = session.NewSession(m.Key)
	m.addRoundTrippers()

	// set the proxy if requested
	raven.CheckError(m.Session.SetProxy(m.GetProxySettings()))
}

// AddModuleCommand adds custom module specific settings and commands to our application
func (m *vimeo) AddModuleCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
}

// Login logs us in for the current session if possible/account available
func (m *vimeo) Login(_ *models.Account) bool {
	return true
}

// Parse parses the tracked item
func (m *vimeo) Parse(item *models.TrackedItem) error {
	if tripper, ok := m.Session.GetClient().Transport.(*vimeoRoundTripper); ok {
		tripper.referer = ""

		originalUrl, err := url.Parse(item.URI)
		if err != nil {
			return err
		}

		parsedQueryString, _ := url.ParseQuery(originalUrl.RawQuery)
		if parsedQueryString.Has("referer") {
			log.WithField("module", m.Key).Debugf(
				"changing referer for video: \"%s\" to \"%s\"",
				item.URI, parsedQueryString.Get("referer"),
			)
			tripper.referer = parsedQueryString.Get("referer")
		}
	} else {
		m.addRoundTrippers()
		return m.Parse(item)
	}

	masterJsonURL := item.URI
	videoTitle := fp.SanitizePath(strconv.Itoa(item.ID), false)
	if !m.masterJsonPattern.MatchString(masterJsonURL) {
		playerJson, err := m.getPlayerJSON(item)
		if err != nil {
			return err
		}

		masterJsonURL = playerJson.GetMasterJSONUrl()
		if masterJsonURL == "" {
			if playerJson.Message != "" {
				return fmt.Errorf("unable to find master.json URL, error message: \"%s\"", playerJson.Message)
			} else {
				return fmt.Errorf("unable to find master.json URL, but no error message could be found")
			}
		}

		videoTitle = fmt.Sprintf(
			"%s_%s_%s",
			playerJson.Video.ID.String(),
			playerJson.Video.Owner.Name,
			fp.SanitizePath(playerJson.GetVideoTitle(), false),
		)
	}

	if masterJsonURL == "" {
		log.WithField("module", m.Key).Warnf(
			"unable to download video from: \"%s\", possibly password protected? skipping", item.URI,
		)
		return nil
	}

	return m.parseVideo(item, masterJsonURL, videoTitle)
}

// AddRoundTrippers adds the round trippers for CloudFlare, adds a custom user agent
// and implements the implicit OAuth2 authentication and sets the Token round tripper
func (m *vimeo) addRoundTrippers() {
	client := m.Session.GetClient()
	// apply CloudFlare bypass
	options := cloudflarebp.GetDefaultOptions()
	client.Transport = cloudflarebp.AddCloudFlareByPass(client.Transport, options)
	client.Transport = m.setVimeoHeaders(client.Transport, "")
}
