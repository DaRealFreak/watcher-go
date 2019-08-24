package tar

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAddFile tests if a file can be added to the archive without errors
// and that the written size equals to the byte size
func TestAddFile(t *testing.T) {
	var assertion = assert.New(t)
	var testFileContent = []byte("123456")

	// create a temporary directory for our tests and to create our files in
	tmpDir, err := ioutil.TempDir("", "")
	assertion.NoError(err)

	// create a new archive
	tmpArchiveFile, err := ioutil.TempFile(tmpDir, "*.tar")
	assertion.NoError(err)
	archive := NewArchive(tmpArchiveFile)

	// add our test content to the archive and check the written size
	written, err := archive.AddFile("testFile", testFileContent)
	assertion.NoError(err)
	assertion.Equal(written, int64(len(testFileContent)))

	// close our archive
	assertion.NoError(archive.Close())
}

// TestAddFile tests if a file can be added to the archive without errors
// and that the written size equals to the byte size
func TestAddFileByPath(t *testing.T) {
	var assertion = assert.New(t)
	var testFileContent = []byte("123456")
	// create a temporary directory for our tests and to create our files in
	tmpDir, err := ioutil.TempDir("", "")
	assertion.NoError(err)

	// generate a temporary .test file
	tmpFile, err := ioutil.TempFile(tmpDir, "*.test")
	assertion.NoError(err)

	// write the test content into the temp file
	writtenTmpFile, err := tmpFile.Write(testFileContent)
	assertion.NoError(err)
	assertion.Equal(writtenTmpFile, len(testFileContent))

	// create a new archive
	tmpArchiveFile, err := ioutil.TempFile(tmpDir, "*.tar")
	assertion.NoError(err)
	archive := NewArchive(tmpArchiveFile)

	// add the created tmp file to the archive and check the written size
	writtenArchive, err := archive.AddFileByPath("test", tmpFile.Name())
	assertion.NoError(err)
	assertion.Equal(writtenArchive, int64(len(testFileContent)))

	// close our archive
	assertion.NoError(archive.Close())
}
