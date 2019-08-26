package gzip

import (
	"archive/tar"
	"compress/gzip"
	"github.com/DaRealFreak/watcher-go/pkg/archive"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	"io"
)

// gzipArchiveReader wrapper for gzip archives to be used as the other archive types
type gzipArchiveReader struct {
	archive.Reader
	tarReader  *tar.Reader
	gzipReader *gzip.Reader
}

// NewReader initializes the readers and returns the struct
func NewReader(f io.Reader) archive.Reader {
	gzipReader, err := gzip.NewReader(f)
	raven.CheckError(err)

	return &gzipArchiveReader{
		tarReader:  tar.NewReader(gzipReader),
		gzipReader: gzipReader,
	}
}

// GetFiles returns all files in the archive
func (a *gzipArchiveReader) GetFiles() (files []string, err error) {
	return files, nil
}

// GetFileContent returns the file content if it exists, returns empty []byte if the file does not exist
func (a *gzipArchiveReader) GetFileContent(fileName string) (content []byte) {
	return content
}

// Close closes the reader
func (a *gzipArchiveReader) Close() (err error) {
	return err
}
