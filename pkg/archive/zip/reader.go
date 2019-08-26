package zip

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"

	"github.com/DaRealFreak/watcher-go/pkg/archive"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
)

// zipArchiveReader wrapper for zip archives to be used as the other archive types
type zipArchiveReader struct {
	archive.Reader
	zipReader *zip.Reader
}

// NewReader initializes the reader and returns the struct
func NewReader(f io.Reader) archive.Reader {
	// we have to convert the io.Reader to an io.ReaderAt for zip files, so copy the whole thing into a new buffer
	buff := bytes.NewBuffer([]byte{})
	size, err := io.Copy(buff, f)
	raven.CheckError(err)
	// and create a bytes reader out of it (which implements all required functions of io.ReaderAt)
	reader := bytes.NewReader(buff.Bytes())
	zipReader, err := zip.NewReader(reader, size)
	raven.CheckError(err)
	return &zipArchiveReader{
		zipReader: zipReader,
	}
}

// GetFiles returns all files in the archive
func (a *zipArchiveReader) GetFiles() (files []string, err error) {
	for _, f := range a.zipReader.File {
		files = append(files, f.Name)
	}
	return files, nil
}

// GetFile returns the reader the for the passed archive file
func (a *zipArchiveReader) GetFile(fileName string) (reader io.Reader, err error) {
	for _, f := range a.zipReader.File {
		if f.Name == fileName {
			file, err := f.Open()
			if err != nil {
				return nil, err
			}
			return file, nil
		}
	}
	return nil, fmt.Errorf("file not found in archive")
}
