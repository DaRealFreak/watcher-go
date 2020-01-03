// Package gdrive contains the implementation of the google drive module
package gdrive

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/pkg/http/session"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/modules"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// gdrive contains the implementation of the ModuleInterface
type gdrive struct {
	*models.Module
	driveService *drive.Service
	settings     gdriveSettings
}

type gdriveSettings struct {
	ServiceAccountFileLocation string `mapstructure:"service_json_path"`
}

// nolint: gochecknoinits
// init function registers the bare and the normal module to the module factories
func init() {
	modules.GetModuleFactory().RegisterModule(NewBareModule())
}

// NewBareModule returns a bare module implementation for the CLI options
func NewBareModule() *models.Module {
	module := &models.Module{
		Key:           "drive.google.com",
		RequiresLogin: false,
		LoggedIn:      true,
		URISchemas: []*regexp.Regexp{
			regexp.MustCompile(`.*drive\.google\.`),
		},
	}
	module.ModuleInterface = &gdrive{
		Module: module,
	}

	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key,
		Color: "1:11",
	})

	return module
}

// InitializeModule initializes the module
func (m *gdrive) InitializeModule() {
	// initialize session
	m.Session = session.NewSession(m.Key)

	// set the proxy if requested
	raven.CheckError(m.Session.SetProxy(m.GetProxySettings()))

	// initialize settings
	raven.CheckError(viper.UnmarshalKey(
		fmt.Sprintf("Modules.%s", m.GetViperModuleKey()),
		&m.settings,
	))

	if _, err := os.Stat(m.settings.ServiceAccountFileLocation); os.IsNotExist(err) {
		log.WithField("module", m.Key).Fatal(
			"google drive requires a service account file to communicate with the Google Drive API",
		)
	}

	// since it technically is still a login but not required we'll call the login function nonetheless
	m.Login(nil)
}

// AddSettingsCommand adds custom module specific settings and commands to our application
func (m *gdrive) AddSettingsCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
}

// Login logs us in for the current session if possible/account available
func (m *gdrive) Login(_ *models.Account) bool {
	absoluteCredentialPath, err := filepath.Abs(m.settings.ServiceAccountFileLocation)
	raven.CheckError(err)

	httpClientContext := context.WithValue(context.Background(), oauth2.HTTPClient, m.Session.GetClient())

	driveService, err := drive.NewService(httpClientContext, option.WithCredentialsFile(absoluteCredentialPath))
	raven.CheckError(err)

	m.driveService = driveService

	return true
}

// Parse parses the tracked item
func (m *gdrive) Parse(item *models.TrackedItem) error {
	list, err := m.driveService.Files.List().
		OrderBy("modifiedTime desc").
		Q(fmt.Sprintf("'%s' in parents", "1xkdljI7kgZ8VZdcCFis2xEtu0iGn3Zy3")).
		Fields("*").
		Do()
	if err != nil {
		panic(err)
	}

	for _, file := range list.Files {
		if file.MimeType == "application/vnd.google-apps.folder" {
			fmt.Println(fmt.Sprintf("folder: %s (id: %s)", file.Name, file.Id))
		} else {
			res, err := m.driveService.Files.Get(file.Id).Download()
			if err != nil {
				return err
			}

			localFile, err := os.Create(file.Name)
			if err != nil {
				return err
			}

			_, err = io.Copy(localFile, res.Body)
			if err != nil {
				return err
			}

			fmt.Println(fmt.Sprintf("downloaded file: %s (id: %s)", file.Name, file.Id))
		}
	}

	return nil
}
