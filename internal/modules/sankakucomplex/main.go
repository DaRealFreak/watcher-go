// Package sankakucomplex contains the implementation of the sankakucomplex module
package sankakucomplex

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/spf13/viper"

	"github.com/DaRealFreak/watcher-go/internal/modules/sankakucomplex/api"

	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/internal/http/session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/spf13/cobra"
)

// sankakuComplex contains the implementation of the ModuleInterface
type sankakuComplex struct {
	*models.Module
	api      *api.SankakuComplexApi
	settings sankakuSettings
}

type sankakuSettings struct {
	Download struct {
		SkipBrokenStreams bool `mapstructure:"skip_broken_streams"`
	} `mapstructure:"download"`
}

// nolint: gochecknoinits
// init function registers the bare and the normal module to the module factories
func init() {
	modules.GetModuleFactory().RegisterModule(NewBareModule())
}

// NewBareModule returns a bare module implementation for the CLI options
func NewBareModule() *models.Module {
	module := &models.Module{
		Key:           "chan.sankakucomplex.com",
		RequiresLogin: false,
		LoggedIn:      false,
		URISchemas: []*regexp.Regexp{
			regexp.MustCompile(".*sankakucomplex.com"),
		},
	}
	module.ModuleInterface = &sankakuComplex{
		Module: module,
	}

	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key,
		Color: "232:208",
	})

	return module
}

// InitializeModule initializes the module
func (m *sankakuComplex) InitializeModule() {
	// initialize settings
	raven.CheckError(viper.UnmarshalKey(
		fmt.Sprintf("Modules.%s", m.GetViperModuleKey()),
		&m.settings,
	))

	// set the module implementation for access to the session, database, etc
	m.Session = session.NewSession(m.Key)

	// set the proxy if requested
	raven.CheckError(m.Session.SetProxy(m.GetProxySettings()))

	m.api = api.NewSankakuComplexApi(m.Key, m.Session, nil)

	m.Initialized = true
}

// AddModuleCommand adds custom module specific settings and commands to our application
func (m *sankakuComplex) AddModuleCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
}

// Login logs us in for the current session if possible/account available
func (m *sankakuComplex) Login(account *models.Account) bool {
	// overwrite our previous API with a logged in instance
	m.api = api.NewSankakuComplexApi(m.Key, m.Session, account)

	m.TriedLogin = true
	m.LoggedIn = m.api.LoginSuccessful()

	return m.LoggedIn
}

// Parse parses the tracked item
func (m *sankakuComplex) Parse(item *models.TrackedItem) error {
	// update sub folder if not set yet
	if item.SubFolder == "" {
		m.DbIO.ChangeTrackedItemSubFolder(item, m.getDownloadTag(item))
	}

	bookPattern := regexp.MustCompile(`/books\?`)
	singleBookPattern := regexp.MustCompile(`/books/(\d+)`)
	wikiPattern := regexp.MustCompile(`/wiki`)

	itemDownloadQueue := &downloadQueue{}

	if wikiPattern.MatchString(item.URI) {
		tagName, err := m.extractItemTag(item)
		if err != nil {
			return err
		}

		bookUri := fmt.Sprintf("https://beta.sankakucomplex.com/books?tags=%s", url.QueryEscape(tagName))
		bookItem := m.DbIO.GetFirstOrCreateTrackedItem(bookUri, "", m)

		galleryUri := fmt.Sprintf("https://beta.sankakucomplex.com/?tags=%s", url.QueryEscape(tagName))
		galleryItem := m.DbIO.GetFirstOrCreateTrackedItem(galleryUri, "", m)

		if err = m.Parse(bookItem); err != nil {
			return err
		}

		return m.Parse(galleryItem)
	}

	if singleBookPattern.MatchString(item.URI) {
		galleryItems, err := m.parseSingleBook(item, singleBookPattern.FindStringSubmatch(item.URI)[1])
		if err != nil {
			return err
		}

		itemDownloadQueue.items = galleryItems
	} else if bookPattern.MatchString(item.URI) {
		bookItems, err := m.parseBooks(item)
		if err != nil {
			return err
		}

		itemDownloadQueue.books = bookItems
	} else {
		galleryItems, err := m.parseGallery(item)
		if err != nil {
			return err
		}

		itemDownloadQueue.items = galleryItems
	}

	return m.processDownloadQueue(itemDownloadQueue, item)
}

func (m *sankakuComplex) AddItem(uri string) (string, error) {
	if parsed, parsedErr := url.Parse(uri); parsedErr == nil {
		queries := parsed.Query()
		if queries.Has("tags") {
			newTagQuery := strings.TrimSpace(strings.ReplaceAll(queries.Get("tags"), "order:popularity", ""))
			queries.Set("tags", newTagQuery)
			parsed.RawQuery = queries.Encode()
		}

		uri = parsed.String()
	}

	return uri, nil
}
