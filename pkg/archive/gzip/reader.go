package gzip

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"

	"github.com/DaRealFreak/watcher-go/pkg/archive"
)

// gzipArchiveReader wrapper for gzip archives to be used as the other archive types
type gzipArchiveReader struct {
	archive.Reader
	buffer    *bytes.Buffer
	tarReader *tar.Reader
}

// NewReader initializes the readers and returns the struct
func NewReader(f io.Reader) archive.Reader {
	buf := new(bytes.Buffer)
	// copy the stream to the buffer
	_, _ = io.Copy(buf, f)
	return &gzipArchiveReader{
		buffer: buf,
	}
}

// GetFiles returns all files in the archive
func (a *gzipArchiveReader) GetFiles() (files []string, err error) {
	if err := a.resetReader(); err != nil {
		return files, err
	}

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
	if err := a.resetReader(); err != nil {
		return reader, err
	}

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

// resetReader recreates the tar reader from the saved buffer
func (a *gzipArchiveReader) resetReader() (err error) {
	gzipReader, err := gzip.NewReader(bytes.NewReader(a.buffer.Bytes()))
	if err != nil {
		return err
	}
	a.tarReader = tar.NewReader(gzipReader)
	return nil
}
