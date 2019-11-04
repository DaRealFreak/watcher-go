package tar

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"

	"github.com/DaRealFreak/watcher-go/pkg/archive"
)

// tarArchiveReader wrapper for tar archives to be used as the other archive types
type tarArchiveReader struct {
	archive.Reader
	buffer    *bytes.Buffer
	tarReader *tar.Reader
}

// NewReader initializes the reader and returns the struct
func NewReader(f io.Reader) archive.Reader {
	buf := new(bytes.Buffer)
	// copy the stream to the buffer
	_, _ = io.Copy(buf, f)

	return &tarArchiveReader{
		buffer: buf,
	}
}

// GetFiles returns all files in the archive
func (a *tarArchiveReader) GetFiles() (files []string, err error) {
	a.resetReader()

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

// HasFile checks if the archive has a file with the passed file path
func (a *tarArchiveReader) HasFile(fileName string) (exists bool, err error) {
	files, err := a.GetFiles()
	if err != nil {
		return false, err
	}

	for _, archivedFileName := range files {
		if fileName == archivedFileName {
			return true, nil
		}
	}

	return false, nil
}

// GetFile returns the reader the for the passed archive file
func (a *tarArchiveReader) GetFile(fileName string) (reader io.Reader, err error) {
	a.resetReader()

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
func (a *tarArchiveReader) resetReader() {
	a.tarReader = tar.NewReader(bytes.NewReader(a.buffer.Bytes()))
}
