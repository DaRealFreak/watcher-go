package tar

import (
	goTar "archive/tar"
	"io"
	"os"
	"time"

	"github.com/DaRealFreak/watcher-go/pkg/archive"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
)

// tarArchive adding both gzip and tar writer
type tarArchive struct {
	archive.Archive
	tarWriter *goTar.Writer
}

// NewArchive initializes the writers and returns the struct
func NewArchive(target io.Writer) archive.Archive {
	return &tarArchive{
		tarWriter: goTar.NewWriter(target),
	}
}

// AddFile adds a file directly from the binary data
func (a *tarArchive) AddFile(name string, fileContent []byte) (writtenSize int64, err error) {
	header := &goTar.Header{
		Typeflag:   goTar.TypeReg,
		Name:       name,
		Linkname:   "",
		Size:       int64(len(fileContent)),
		Mode:       0644,
		ModTime:    time.Now(),
		AccessTime: time.Now(),
		ChangeTime: time.Now(),
		Format:     goTar.FormatPAX,
	}

	// write the header to the tar
	if err = a.tarWriter.WriteHeader(header); err != nil {
		return 0, err
	}

	writtenSizeInt, err := a.tarWriter.Write(fileContent)
	return int64(writtenSizeInt), err
}

// AddFileByPath adds a file which he tries to read from a local path
func (a *tarArchive) AddFileByPath(name string, filePath string) (writtenSize int64, err error) {
	// open the file and defer closing it
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer raven.CheckReadCloser(file)

	// retrieve file stats for headers
	info, err := file.Stat()
	if err != nil {
		return 0, err
	}

	// create the tar header and write it
	header, err := goTar.FileInfoHeader(info, name)
	if err != nil {
		return 0, err
	}

	// set the name of the header again
	header.Name = name

	// write the header to the tar
	if err = a.tarWriter.WriteHeader(header); err != nil {
		return 0, err
	}

	writtenSize, err = io.Copy(a.tarWriter, file)
	return writtenSize, err
}

// Close closes the writers of the archive
func (a *tarArchive) Close() error {
	return a.tarWriter.Close()
}
