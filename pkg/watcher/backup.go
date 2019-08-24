package watcher

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/DaRealFreak/watcher-go/pkg/archive"
	"github.com/DaRealFreak/watcher-go/pkg/archive/gzip"
	"github.com/DaRealFreak/watcher-go/pkg/archive/tar"
	"github.com/DaRealFreak/watcher-go/pkg/archive/zip"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
)

// BackupEverything backs up the full database and the configuration
func (app *Watcher) BackupEverything(archiveName string, cfg *AppConfiguration) {
	writer, err := app.getArchiveWriter(archiveName, cfg)
	raven.CheckError(err)
	_, err = writer.AddFileByPath("watcher.db", "watcher.db")
	raven.CheckError(err)

	err = writer.Close()
	fmt.Println(err)
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
	case cfg.Backup.Gzip, cfg.Backup.Tar && cfg.Backup.Zip:
		return gzip.FileExt
	case cfg.Backup.Tar:
		return tar.FileExt
	case cfg.Backup.Zip:
		return zip.FileExt
	default:
		// not directly passed archive type, use zip on windows, gzip on other systems
		if runtime.GOOS == "windows" {
			return zip.FileExt
		}
		return gzip.FileExt
	}
}
