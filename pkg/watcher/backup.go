package watcher

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/DaRealFreak/watcher-go/pkg/archive"
	"github.com/DaRealFreak/watcher-go/pkg/archive/gzip"
	"github.com/DaRealFreak/watcher-go/pkg/archive/tar"
	"github.com/DaRealFreak/watcher-go/pkg/archive/zip"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	"github.com/spf13/viper"
)

// Backup backs up the full database and the configuration
func (app *Watcher) Backup(archiveName string, cfg *AppConfiguration) {
	writer, err := app.getArchiveWriter(archiveName, cfg)
	raven.CheckError(err)

	if cfg.Backup.Settings {
		raven.CheckError(app.backupSettings(writer, cfg))
	}

	if cfg.Backup.Database.Accounts.Enabled || cfg.Backup.Database.Items.Enabled {
		raven.CheckError(app.backupDatabase(writer, cfg))
	}

	raven.CheckError(writer.Close())
}

// backupDatabase generates an SQL file for items/accounts and adds it to the archive
// if items and accounts are exported and SQL mode is not active we just archive the db file
func (app *Watcher) backupDatabase(writer archive.Archive, cfg *AppConfiguration) (err error) {
	switch {
	case cfg.Backup.Database.Accounts.Enabled && cfg.Backup.Database.Items.Enabled:
		if cfg.Backup.Database.SQL {
			fmt.Println("ToDo export full database")
		} else {
			_, err = writer.AddFileByPath(
				path.Base(viper.GetString("Database.Path")),
				viper.GetString("Database.Path"),
			)
		}
	case cfg.Backup.Database.Accounts.Enabled:
		fmt.Println("ToDo export accounts")
	case cfg.Backup.Database.Items.Enabled:
		fmt.Println("ToDo export items")
	}
	return err
}

// backupSettings adds the setting file to the archive
func (app *Watcher) backupSettings(writer archive.Archive, cfg *AppConfiguration) (err error) {
	settingsPath := cfg.ConfigurationFile
	if settingsPath == "" {
		settingsPath = "./.watcher.yaml"
	}
	_, err = writer.AddFileByPath(
		path.Base(settingsPath),
		settingsPath,
	)
	return err
}

// getArchiveWriter returns the used archive based on the passed app configuration
func (app *Watcher) getArchiveWriter(archiveName string, cfg *AppConfiguration) (writer archive.Archive, err error) {
	var archiveWriter archive.Archive

	// retrieve the archive extension type and attach it if not already set by the user
	archiveExt := app.getArchiveExtension(cfg)
	if !strings.HasSuffix(archiveName, archiveExt) {
		archiveName += archiveExt
	}

	// create our archive
	f, err := os.Create(archiveName)
	if err != nil {
		return nil, err
	}

	// retrieve the archive writer based on the archive extension
	switch archiveExt {
	case gzip.FileExt:
		archiveWriter = gzip.NewArchive(f)
	case tar.FileExt:
		archiveWriter = tar.NewArchive(f)
	case zip.FileExt:
		archiveWriter = zip.NewArchive(f)
	}
	return archiveWriter, nil
}

// getArchiveExtension returns the archive extension based on the app configuration
func (app *Watcher) getArchiveExtension(cfg *AppConfiguration) (ext string) {
	switch {
	case cfg.Backup.Archive.Gzip, cfg.Backup.Archive.Tar && cfg.Backup.Archive.Zip:
		return gzip.FileExt
	case cfg.Backup.Archive.Tar:
		return tar.FileExt
	case cfg.Backup.Archive.Zip:
		return zip.FileExt
	default:
		// not directly passed archive type, use zip on windows, gzip on other systems
		if runtime.GOOS == "windows" {
			return zip.FileExt
		}
		return gzip.FileExt
	}
}
