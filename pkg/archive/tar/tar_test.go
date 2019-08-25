package tar

import (
	"io/ioutil"
	"testing"

	"github.com/DaRealFreak/watcher-go/pkg/archive/archivetest"
	"github.com/stretchr/testify/assert"
)

// TestAddFile tests if a file can be added to the archive without errors
// and that the written size equals to the byte size
func TestAddFile(t *testing.T) {
	var assertion = assert.New(t)
	// create a new archive
	tmpArchiveFile, err := ioutil.TempFile("", "*"+FileExt)
	assertion.NoError(err)
	archive := NewArchive(tmpArchiveFile)
	// run the test for the gzip implementation
	archivetest.AddFile(archive, t)
}

// TestAddFile tests if a file can be added to the archive without errors
// and that the written size equals to the byte size
func TestAddFileByPath(t *testing.T) {
	var assertion = assert.New(t)
	// create a new archive
	tmpArchiveFile, err := ioutil.TempFile("", "*"+FileExt)
	assertion.NoError(err)
	archive := NewArchive(tmpArchiveFile)
	// run the test for the gzip implementation
	archivetest.AddFileByPath(archive, t)
}
