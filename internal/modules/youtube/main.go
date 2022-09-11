package kemono

import (
	"bytes"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/internal/http/session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type youtube struct {
	*models.Module
	settings youtubeSettings
}

type youtubeSettings struct {
	CookieFileLocation  string `mapstructure:"cookie_file_location"`
	ArchiveFileLocation string `mapstructure:"archive_file_location"`
}

// init function registers the bare and the normal module to the module factories
func init() {
	modules.GetModuleFactory().RegisterModule(NewBareModule())
}

// NewBareModule returns a bare module implementation for the CLI options
func NewBareModule() *models.Module {
	module := &models.Module{
		Key:           "youtube.com",
		RequiresLogin: false,
		LoggedIn:      false,
		URISchemas: []*regexp.Regexp{
			regexp.MustCompile(`youtube.com`),
			regexp.MustCompile(`youtu.be`),
		},
	}
	module.ModuleInterface = &youtube{
		Module: module,
	}

	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key,
		Color: "232:160",
	})

	return module
}

// InitializeModule initializes the module
func (m *youtube) InitializeModule() {
	// initialize settings
	raven.CheckError(viper.UnmarshalKey(
		fmt.Sprintf("Modules.%s", m.GetViperModuleKey()),
		&m.settings,
	))

	if m.settings.CookieFileLocation != "" {
		if _, err := os.Stat(m.settings.CookieFileLocation); os.IsNotExist(err) {
			log.WithField("module", m.Key).Fatal(
				"the cookie file could not be stated but could not be read",
			)
		}
	} else {
		log.WithField("module", m.Key).Warn(
			"no cookie file location is set, potentially unable to download private/adult videos",
		)
	}

	if m.settings.ArchiveFileLocation != "" {
		if _, err := os.Stat(m.settings.ArchiveFileLocation); os.IsNotExist(err) {
			// create archive file if it doesn't exist yet
			raven.CheckError(os.MkdirAll(filepath.Dir(m.settings.ArchiveFileLocation), fs.ModeType))
			_, err = os.Create(m.settings.ArchiveFileLocation)
			raven.CheckError(err)
		}
	} else {
		log.WithField("module", m.Key).Warn(
			"no archive file location is set, it'll download videos of lists and users multiple times",
		)
	}

	// set the module implementation for access to the session, database, etc
	m.Session = session.NewSession(m.Key)

	// set the proxy if requested
	raven.CheckError(m.Session.SetProxy(m.GetProxySettings()))

	m.Initialized = true
}

// AddModuleCommand adds custom module specific settings and commands to our application
func (m *youtube) AddModuleCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
}

// Login logs us in for the current session if possible/account available
func (m *youtube) Login(_ *models.Account) bool {
	return true
}

// Parse parses the tracked item
func (m *youtube) Parse(item *models.TrackedItem) error {
	parsedUrl, parseErr := url.Parse(item.URI)
	if parseErr != nil {
		return parseErr
	}

	args, cmdErr := m.getCommand(item)
	if cmdErr != nil {
		return cmdErr
	}

	log.Debugf("running command: yt-dlp %s", strings.Join(args, " "))
	_, stderr, err := executeCommand(exec.Command("yt-dlp", args...))
	if stderr.Len() > 0 {
		return fmt.Errorf("running command returned error: %s", stderr.String())
	}

	if err != nil {
		return err
	}

	// single video, we can set this one to complete
	if parsedUrl.Query().Has("v") && !parsedUrl.Query().Has("list") {
		m.DbIO.ChangeTrackedItemCompleteStatus(item, true)
	}

	return nil
}

func (m *youtube) getCommand(item *models.TrackedItem) (args []string, err error) {
	localFolder := filepath.Join(
		viper.GetString("download.directory"),
		m.Key,
	)
	if item.SubFolder != "" {
		localFolder = filepath.Join(
			localFolder,
			fp.TruncateMaxLength(fp.SanitizePath(item.SubFolder, false)),
		)
	} else {
		localFolder = filepath.Join(
			localFolder,
			"%(uploader)s",
		)
	}

	args = []string{"-f", "bestvideo[ext=mp4]+bestaudio[ext=m4a]/bestvideo+bestaudio",
		"--merge-output-format", "mp4",
		"-ciw",
		"-o",
		localFolder + "/%(title)s - (%(id)s).%(ext)s",
	}

	if _, err = os.Stat(m.settings.ArchiveFileLocation); err == nil {
		absPath, absErr := filepath.Abs(m.settings.ArchiveFileLocation)
		if absErr != nil {
			return args, absErr
		}

		args = append(args, "--download-archive", absPath)
	}

	if _, err = os.Stat(m.settings.CookieFileLocation); err == nil {
		absPath, absErr := filepath.Abs(m.settings.CookieFileLocation)
		if absErr != nil {
			return args, absErr
		}

		args = append(args, "--cookies", absPath)
	}

	usedProxy := m.GetProxySettings()

	// set proxy and overwrite the client if the proxy is enabled
	if usedProxy != nil && usedProxy.Enable {
		args = append(args, "--proxy", usedProxy.GetProxyString())
	}

	// append the YouTube URL at the end
	args = append(args, item.URI)

	return args, err
}

// executeCommand the command and returns the output/error
func executeCommand(cmd *exec.Cmd) (stdout bytes.Buffer, stderr bytes.Buffer, err error) {
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Start()
	if err != nil {
		return stdout, stderr, err
	}

	err = cmd.Wait()

	return stdout, stderr, err
}
