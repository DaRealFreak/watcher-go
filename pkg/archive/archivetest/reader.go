// Package archivetest contains the shared testing utility of all archive implementations
package archivetest

import (
	"io/ioutil"
	"testing"

	"github.com/DaRealFreak/watcher-go/pkg/archive"
	"github.com/stretchr/testify/assert"
)

// GetFiles checks if all added files are contained in the archive
func GetFiles(t *testing.T, reader archive.Reader) {
	assertion := assert.New(t)

	archiveFiles, err := reader.GetFiles()
	assertion.NoError(err)
	assertion.Equal(len(GetSharedTestFiles()), len(archiveFiles))
}

// GetFile tests if all generated files can get retrieved from the generated archive
func GetFile(t *testing.T, reader archive.Reader) {
	assertion := assert.New(t)

	for fileName, fileContent := range GetSharedTestFiles() {
		reader, err := reader.GetFile(fileName)
		assertion.NoError(err)
		content, err := ioutil.ReadAll(reader)
		assertion.NoError(err)
		assertion.Equal(content, []byte(fileContent))
	}
}

// GenerateTestFiles generates a new archive with the passed files
func GenerateTestFiles(t *testing.T, writer archive.Writer) {
	var assertion = assert.New(t)

	for fileName, fileContent := range GetSharedTestFiles() {
		_, err := writer.AddFile(fileName, []byte(fileContent))
		assertion.NoError(err)
	}

	err := writer.Close()
	assertion.NoError(err)
}

// GetSharedTestFiles returns the shared test files/file content for the unit tests
func GetSharedTestFiles() map[string]string {
	return map[string]string{
		"accounts.sql":        "INSERT INTO test; COMMIT;",
		"README":              "some meaningful content",
		"content_dir/LICENSE": "license content",
		// FixMe: include empty dir cases (works with zip, not with tar)
		// "empty_dir/":          "",
	}
}
