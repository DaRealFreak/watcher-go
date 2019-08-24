package gzip

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"time"

	"github.com/DaRealFreak/watcher-go/pkg/archive"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
)

// gzipArchive adding both gzip and tar writer
type gzipArchive struct {
	archive.Archive
	gzipWriter *gzip.Writer
	tarWriter  *tar.Writer
}

// NewArchive initializes the writers and returns the struct
func NewArchive(target io.Writer) archive.Archive {
	writer := &gzipArchive{
		gzipWriter: gzip.NewWriter(target),
	}
	writer.tarWriter = tar.NewWriter(writer.gzipWriter)
	return writer
}

// AddFile adds a file directly from the binary data
func (a *gzipArchive) AddFile(name string, fileContent []byte) (writtenSize int64, err error) {
	header := &tar.Header{
		Typeflag:   tar.TypeReg,
		Name:       name,
		Linkname:   "",
		Size:       int64(len(fileContent)),
		Mode:       0644,
		ModTime:    time.Now(),
		AccessTime: time.Now(),
		ChangeTime: time.Now(),
		Format:     tar.FormatPAX,
	}

	// write the header to the tar
	if err = a.tarWriter.WriteHeader(header); err != nil {
		return 0, err
	}

	writtenSizeInt, err := a.tarWriter.Write(fileContent)
	return int64(writtenSizeInt), err
}

// AddFileByPath adds a file which he tries to read from a local path
func (a *gzipArchive) AddFileByPath(name string, filePath string) (writtenSize int64, err error) {
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
	header, err := tar.FileInfoHeader(info, name)
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
func (a *gzipArchive) Close() error {
	if err := a.tarWriter.Close(); err != nil {
		return err
	}
	return a.gzipWriter.Close()
}