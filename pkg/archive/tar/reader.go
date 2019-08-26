package tar

import (
	"archive/tar"
	"github.com/DaRealFreak/watcher-go/pkg/archive"
	"io"
)

// tarArchiveReader wrapper for tar archives to be used as the other archive types
type tarArchiveReader struct {
	archive.Reader
	tarReader *tar.Reader
}

// NewReader initializes the reader and returns the struct
func NewReader(f io.Reader) archive.Reader {
	return &tarArchiveReader{
		tarReader: tar.NewReader(f),
	}
}

// GetFiles returns all files in the archive
func (a *tarArchiveReader) GetFiles() (files []string, err error) {
	return files, nil
}

// GetFile returns the reader the for the passed archive file
func (a *tarArchiveReader) GetFile(fileName string) (reader io.Reader, err error) {
	return nil, nil
}

// Close closes the reader
func (a *tarArchiveReader) Close() (err error) {
	return err
}
