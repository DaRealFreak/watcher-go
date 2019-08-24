package archive

// Archive is the interface for all valid archive types (zip, gzip, tar)
type Archive interface {
	AddFile(name string, fileContent []byte) (writtenSize int64, err error)
	AddFileByPath(name string, filePath string) (writtenSize int64, err error)
	Close() error
}
