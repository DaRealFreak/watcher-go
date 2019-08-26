package gzip

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"

	"github.com/DaRealFreak/watcher-go/pkg/archive"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
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
	for {
		hdr, err := a.tarReader.Next()
		if err == io.EOF {
			// end of archive
			break
		}
		if err != nil {
			return nil, err
		}
		files = append(files, hdr.Name)
	}
	return files, nil
}

// GetFile returns the reader the for the passed archive file
func (a *gzipArchiveReader) GetFile(fileName string) (reader io.Reader, err error) {
	for {
		hdr, err := a.tarReader.Next()
		if err == io.EOF {
			// end of archive
			break
		}
		if err != nil {
			return nil, err
		}

		if hdr.Name == fileName {
			return a.tarReader, nil
		}
	}
	return nil, fmt.Errorf("file not found in archive")
}
