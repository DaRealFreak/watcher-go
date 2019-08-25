package watcher

import (
	"fmt"
	"os"

	"github.com/DaRealFreak/watcher-go/pkg/archive"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
)

// Restore restores the database/settings from the passed archive
func (app *Watcher) Restore(archiveName string, cfg *AppConfiguration) {
	reader, err := app.getArchiveReader(archiveName)
	raven.CheckError(err)

	if cfg.Backup.Settings {
		raven.CheckError(app.restoreSettings(reader, cfg))
	}

	if cfg.Backup.Database.Accounts.Enabled || cfg.Backup.Database.Items.Enabled {
		raven.CheckError(app.restoreDatabase(reader, cfg))
	}

	raven.CheckError(reader.Close())
}

// getArchiveReader returns the reader for the passed archive if it exists and can be opened
func (app *Watcher) getArchiveReader(archiveName string) (reader archive.Reader, err error) {
	// #nosec
	_, err = os.Open(archiveName)
	if err != nil {
		return nil, err
	}
	return reader, err
}

// restoreDatabase checks
// if items and accounts are exported and SQL mode is not active we just archive the db file
func (app *Watcher) restoreDatabase(writer archive.Reader, cfg *AppConfiguration) (err error) {
	switch {
	case cfg.Backup.Database.Accounts.Enabled && cfg.Backup.Database.Items.Enabled:
		// check if [database] exists in archive, else check for accounts.sql and tracked_items.sql
		fmt.Println("ToDo full database restoration")
	case cfg.Backup.Database.Accounts.Enabled:
		// check for accounts.sql in archive
		fmt.Println("ToDo database accounts restoration")
	case cfg.Backup.Database.Items.Enabled:
		// check for tracked_items.sql in archive
		fmt.Println("ToDo database tracked items restoration")
	}
	return err
}

// restoreSettings restores the settings from the archive
func (app *Watcher) restoreSettings(writer archive.Reader, cfg *AppConfiguration) (err error) {
	// check for [cfg.ConfigurationFile] in archive
	return err
}
