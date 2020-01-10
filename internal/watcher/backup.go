package watcher

import (
	"bytes"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/DaRealFreak/watcher-go/pkg/archive"
	"github.com/DaRealFreak/watcher-go/pkg/archive/gzip"
	"github.com/DaRealFreak/watcher-go/pkg/archive/tar"
	"github.com/DaRealFreak/watcher-go/pkg/archive/zip"
	"github.com/spf13/viper"
)

// Backup backs up the full database and the configuration
func (app *Watcher) Backup(archiveName string, cfg *AppConfiguration) {
	writer, err := app.getArchiveWriter(archiveName, cfg)
	raven.CheckError(err)

	if cfg.Backup.Settings {
		raven.CheckError(app.backupSettings(writer, cfg))
	}

	if cfg.Backup.Database.Accounts.Enabled ||
		cfg.Backup.Database.Items.Enabled ||
		cfg.Backup.Database.OAuth2Clients.Enabled ||
		cfg.Backup.Database.Cookies.Enabled {
		raven.CheckError(app.backupDatabase(writer, cfg))
	}

	raven.CheckError(writer.Close())
}

// backupDatabase generates an SQL file for items/accounts and adds it to the archive
// if items and accounts are exported and SQL mode is not active we just archive the db file
func (app *Watcher) backupDatabase(writer archive.Writer, cfg *AppConfiguration) (err error) {
	switch {
	case cfg.Backup.Database.Accounts.Enabled &&
		cfg.Backup.Database.Items.Enabled &&
		cfg.Backup.Database.OAuth2Clients.Enabled &&
		cfg.Backup.Database.Cookies.Enabled:
		if cfg.Backup.Database.SQL {
			for _, table := range []string{"accounts", "tracked_items", "oauth_clients", "cookies"} {
				app.backupTableAsSQL(writer, table)
			}
		} else {
			_, err = writer.AddFileByPath(
				path.Base(viper.GetString("Database.Path")),
				viper.GetString("Database.Path"),
			)
		}
	case cfg.Backup.Database.Accounts.Enabled:
		app.backupTableAsSQL(writer, "accounts")
	case cfg.Backup.Database.Items.Enabled:
		app.backupTableAsSQL(writer, "tracked_items")
	case cfg.Backup.Database.OAuth2Clients.Enabled:
		app.backupTableAsSQL(writer, "oauth_clients")
	case cfg.Backup.Database.Cookies.Enabled:
		app.backupTableAsSQL(writer, "cookies")
	}

	return err
}

// backupTableAsSQL generates an .sql file from the passed table and adds it to the passed archive as [table].sql
func (app *Watcher) backupTableAsSQL(writer archive.Writer, table string) {
	buffer := new(bytes.Buffer)
	raven.CheckError(app.DbCon.DumpTables(buffer, table))

	content, err := ioutil.ReadAll(buffer)
	raven.CheckError(err)

	_, err = writer.AddFile(table+".sql", content)
	raven.CheckError(err)
}

// backupSettings adds the setting file to the archive
func (app *Watcher) backupSettings(writer archive.Writer, cfg *AppConfiguration) (err error) {
	_, err = writer.AddFileByPath(
		path.Base(cfg.ConfigurationFile),
		cfg.ConfigurationFile,
	)

	return err
}

// getArchiveWriter returns the used archive based on the passed app configuration
func (app *Watcher) getArchiveWriter(archiveName string, cfg *AppConfiguration) (writer archive.Writer, err error) {
	var archiveWriter archive.Writer

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
		archiveWriter = gzip.NewWriter(f)
	case tar.FileExt:
		archiveWriter = tar.NewWriter(f)
	case zip.FileExt:
		archiveWriter = zip.NewWriter(f)
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
