package archivetest

import (
	"io/ioutil"
	"testing"

	"github.com/DaRealFreak/watcher-go/pkg/archive"
	"github.com/stretchr/testify/assert"
)

// AddFile tests if a file can be added to the archive without errors
// and that the written size equals to the byte size
func AddFile(archive archive.Archive, t *testing.T) {
	var assertion = assert.New(t)
	var testFileContent = []byte("123456")
	// add our test content to the archive and check the written size
	written, err := archive.AddFile("testFile", testFileContent)
	assertion.NoError(err)
	assertion.Equal(written, int64(len(testFileContent)))

	// close our archive
	assertion.NoError(archive.Close())
}

// AddFileByPath tests if a file can be added to the archive without errors
// and that the written size equals to the byte size
func AddFileByPath(archive archive.Archive, t *testing.T) {
	var assertion = assert.New(t)
	var testFileContent = []byte("123456")

	// generate a temporary .test file
	tmpFile, err := ioutil.TempFile("", "*.test")
	assertion.NoError(err)

	// write the test content into the temp file
	writtenTmpFile, err := tmpFile.Write(testFileContent)
	assertion.NoError(err)
	assertion.Equal(writtenTmpFile, len(testFileContent))

	// add the created tmp file to the archive and check the written size
	writtenArchive, err := archive.AddFileByPath("test", tmpFile.Name())
	assertion.NoError(err)
	assertion.Equal(writtenArchive, int64(len(testFileContent)))

	// close our archive
	assertion.NoError(archive.Close())
}
