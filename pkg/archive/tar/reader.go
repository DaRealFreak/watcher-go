package tar

import (
	"archive/tar"
	"bytes"
	"fmt"
	"github.com/DaRealFreak/watcher-go/pkg/archive"
	"io"
)

// tarArchiveReader wrapper for tar archives to be used as the other archive types
type tarArchiveReader struct {
	archive.Reader
	buffer *bytes.Buffer
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
	// create new tar reader, since the reader can't remember previous bytes on multiple usages
	tarReader := tar.NewReader(bytes.NewReader(a.buffer.Bytes()))
	for {
		hdr, err := tarReader.Next()
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
func (a *tarArchiveReader) GetFile(fileName string) (reader io.Reader, err error) {
	// create new tar reader, since the reader can't remember previous bytes on multiple usages
	tarReader := tar.NewReader(bytes.NewReader(a.buffer.Bytes()))
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			// end of archive
			break
		}
		if err != nil {
			return nil, err
		}

		if hdr.Name == fileName {
			return tarReader, nil
		}
	}
	return nil, fmt.Errorf("file not found in archive")
}
