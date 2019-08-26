package zip

import (
	"archive/zip"
	"bytes"
	"github.com/DaRealFreak/watcher-go/pkg/archive"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	"io"
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

// GetFileContent returns the file content if it exists, returns empty []byte if the file does not exist
func (a *zipArchiveReader) GetFileContent(fileName string) (content []byte) {
	return content
}

// Close closes the reader
func (a *zipArchiveReader) Close() (err error) {
	return err
}
