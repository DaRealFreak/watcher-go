package watcher

import (
	"fmt"
	"github.com/DaRealFreak/watcher-go/pkg/archive/gzip"
	"github.com/DaRealFreak/watcher-go/pkg/archive/tar"
	"github.com/DaRealFreak/watcher-go/pkg/archive/zip"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/DaRealFreak/watcher-go/pkg/archive"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
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
func (app *Watcher) Restore(archiveName string, cfg *AppConfiguration) {
	reader, err := app.getArchiveReader(archiveName)
	raven.CheckError(err)

	if cfg.Restore.Settings {
		raven.CheckError(app.restoreSettings(reader, cfg))
		log.Info("settings restored successfully")
	}

	if cfg.Restore.Database.Accounts.Enabled || cfg.Restore.Database.Items.Enabled {
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
func (app *Watcher) restoreDatabase(reader archive.Reader, cfg *AppConfiguration) (err error) {
	switch {
	case cfg.Restore.Database.Accounts.Enabled && cfg.Restore.Database.Items.Enabled:
		// check if [database] exists in archive, else check for accounts.sql and tracked_items.sql
		fmt.Println("ToDo full database restoration")
	case cfg.Restore.Database.Accounts.Enabled:
		// check for accounts.sql in archive
		fmt.Println("ToDo database accounts restoration")
	case cfg.Restore.Database.Items.Enabled:
		// check for tracked_items.sql in archive
		fmt.Println("ToDo database tracked items restoration")
	}
	return err
}

// restoreSettings restores the settings from the archive
func (app *Watcher) restoreSettings(reader archive.Reader, cfg *AppConfiguration) (err error) {
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
	} else {
		log.Warnf(
			"the passed archive does not contain your current configuration file %s",
			cfg.ConfigurationFile,
		)
	}
	return nil
}
