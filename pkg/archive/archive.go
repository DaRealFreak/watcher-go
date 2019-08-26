package archive

import "io"

// Writer is the writer interface for all valid archive types (zip, gzip, tar)
type Writer interface {
	AddFile(name string, fileContent []byte) (writtenSize int64, err error)
	AddFileByPath(name string, filePath string) (writtenSize int64, err error)
	Close() error
}

// Reader is the reader interface for all valid archive types (zip, gzip, tar)
type Reader interface {
	GetFiles() (files []string, err error)
	GetFile(fileName string) (reader io.Reader, err error)
}
