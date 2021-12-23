package watcher

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/configuration"

	"github.com/DaRealFreak/watcher-go/pkg/archive/gzip"
	"github.com/DaRealFreak/watcher-go/pkg/archive/tar"
	"github.com/DaRealFreak/watcher-go/pkg/archive/zip"
	log "github.com/sirupsen/logrus"

	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/DaRealFreak/watcher-go/pkg/archive"
)

// NoReaderFoundError is the error in case the file has no recognized extension
type NoReaderFoundError struct {
	archive string
}

// Error prints the error details for our custom error
func (e NoReaderFoundError) Error() string {
	return fmt.Sprintf("No reader found for archive %s", e.archive)
}

// Restore restores the database/settings from the passed archive
func (app *Watcher) Restore(archiveName string, cfg *configuration.AppConfiguration) {
	reader, err := app.getArchiveReader(archiveName)
	raven.CheckError(err)

	if cfg.Restore.Settings {
		raven.CheckError(app.restoreSettings(reader, cfg))
		log.Info("settings restored successfully")
	}

	if cfg.Restore.Database.Accounts.Enabled ||
		cfg.Restore.Database.Items.Enabled ||
		cfg.Restore.Database.OAuth2Clients.Enabled ||
		cfg.Restore.Database.Cookies.Enabled {
		raven.CheckError(app.restoreDatabase(reader, cfg))
	}
}

// getArchiveReader returns the reader for the passed archive if it exists and can be opened
func (app *Watcher) getArchiveReader(archiveName string) (reader archive.Reader, err error) {
	// #nosec
	file, err := os.Open(archiveName)
	raven.CheckError(err)

	switch {
	case strings.HasSuffix(archiveName, zip.FileExt):
		return zip.NewReader(file), nil
	case strings.HasSuffix(archiveName, tar.FileExt):
		return tar.NewReader(file), nil
	case strings.HasSuffix(archiveName, gzip.FileExt):
		return gzip.NewReader(file), nil
	default:
		return nil, NoReaderFoundError{archive: archiveName}
	}
}

// restoreDatabase checks
// if items and accounts are exported and SQL mode is not active we just archive the db file
func (app *Watcher) restoreDatabase(reader archive.Reader, cfg *configuration.AppConfiguration) (err error) {
	switch {
	case cfg.Restore.Database.Accounts.Enabled &&
		cfg.Restore.Database.Items.Enabled &&
		cfg.Restore.Database.OAuth2Clients.Enabled &&
		cfg.Restore.Database.Cookies.Enabled:
		// check if [database] exists in archive, else check for accounts.sql and tracked_items.sql
		if exists, _ := reader.HasFile(filepath.Base(cfg.Database)); exists {
			file, err := reader.GetFile(filepath.Base(cfg.Database))
			if err != nil {
				return err
			}

			content, err := ioutil.ReadAll(file)
			if err != nil {
				return err
			}

			err = ioutil.WriteFile(cfg.Database, content, os.ModePerm)
			if err != nil {
				return err
			}

			log.Info("restored database file from archive")
		}

		return app.restoreTablesFromArchive(
			reader,
			"accounts.sql", "tracked_items.sql", "oauth_clients.sql", "cookies.sql",
		)
	case cfg.Restore.Database.Accounts.Enabled:
		// check for accounts.sql in archive
		return app.restoreTablesFromArchive(reader, "accounts.sql")
	case cfg.Restore.Database.Items.Enabled:
		// check for tracked_items.sql in archive
		return app.restoreTablesFromArchive(reader, "tracked_items.sql")
	case cfg.Restore.Database.OAuth2Clients.Enabled:
		// check for oauth_clients.sql in archive
		return app.restoreTablesFromArchive(reader, "oauth_clients.sql")
	case cfg.Restore.Database.Cookies.Enabled:
		// check for cookies.sql in archive
		return app.restoreTablesFromArchive(reader, "cookies.sql")
	}

	// no restore option selected, should be unreachable from the command line options
	log.Warning("no restore option selected")

	return nil
}

// restoreTablesFromArchive uses the RestoreTableFromFile func of the database interface
// to import tables from the archived sql files
func (app *Watcher) restoreTablesFromArchive(reader archive.Reader, filesNames ...string) error {
	for _, sqlFileName := range filesNames {
		if exists, _ := reader.HasFile(sqlFileName); exists {
			file, err := ioutil.TempFile("", "*.sql")
			if err != nil {
				return err
			}

			reader, err := reader.GetFile(sqlFileName)
			if err != nil {
				return err
			}

			if _, err := io.Copy(file, reader); err != nil {
				return err
			}

			if err := app.DbCon.RestoreTableFromFile(file.Name()); err != nil {
				return err
			}

			log.Infof("restored database settings from file %s", sqlFileName)
		}
	}

	return nil
}

// restoreSettings restores the settings from the archive
func (app *Watcher) restoreSettings(reader archive.Reader, cfg *configuration.AppConfiguration) (err error) {
	// check for [cfg.ConfigurationFile] in archive
	exists, err := reader.HasFile(filepath.Base(cfg.ConfigurationFile))
	if err != nil {
		return err
	}

	if exists {
		file, err := reader.GetFile(filepath.Base(cfg.ConfigurationFile))
		if err != nil {
			return err
		}

		content, err := ioutil.ReadAll(file)
		if err != nil {
			return err
		}

		return ioutil.WriteFile(cfg.ConfigurationFile, content, os.ModePerm)
	}

	log.Warnf(
		"the passed archive does not contain your current configuration file %s",
		cfg.ConfigurationFile,
	)

	return nil
}
