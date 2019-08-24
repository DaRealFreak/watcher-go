package zip

import (
	"archive/zip"
	"io"
	"os"
	"time"

	"github.com/DaRealFreak/watcher-go/pkg/archive"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
)

// FileExt is the file extension for zip archives
const FileExt = ".zip"

// zipArchive adds a zip writer
type zipArchive struct {
	archive.Archive
	zipWriter *zip.Writer
}

// NewArchive initializes the writers and returns the struct
func NewArchive(target io.Writer) archive.Archive {
	return &zipArchive{
		zipWriter: zip.NewWriter(target),
	}
}

// AddFile adds a file directly from the binary data
func (a *zipArchive) AddFile(name string, fileContent []byte) (writtenSize int64, err error) {
	header := &zip.FileHeader{
		Name:               name,
		Modified:           time.Now(),
		UncompressedSize64: uint64(len(fileContent)),
	}
	header.Method = zip.Deflate
	header.SetMode(os.ModePerm)
	writer, err := a.zipWriter.CreateHeader(header)
	if err != nil {
		return 0, err
	}

	writtenSizeInt, err := writer.Write(fileContent)
	return int64(writtenSizeInt), err
}

// AddFileByPath adds a file which he tries to read from a local path
func (a *zipArchive) AddFileByPath(name string, filePath string) (writtenSize int64, err error) {
	// open the file and defer closing it
	// #nosec
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

	// create the zip header and write it
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return 0, err
	}
	header.Name = name
	header.Method = zip.Deflate

	// retrieve the writer from the header
	w, err := a.zipWriter.CreateHeader(header)
	if err != nil {
		return 0, err
	}
	writtenSize, err = io.Copy(w, file)
	return writtenSize, err
}

// Close closes the writers of the archive
func (a *zipArchive) Close() error {
	return a.zipWriter.Close()
}
