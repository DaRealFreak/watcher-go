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

// GetFile returns the reader the for the passed archive file
func (a *gzipArchiveReader) GetFile(fileName string) (reader io.Reader, err error) {
	return nil, err
}

// Close closes the reader
func (a *gzipArchiveReader) Close() (err error) {
	return err
}
