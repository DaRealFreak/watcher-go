package zip

import (
	"os"
	"testing"

	"github.com/DaRealFreak/watcher-go/pkg/archive/archivetest"
	"github.com/stretchr/testify/assert"
)

// TestAddFile tests if a file can be added to the archive without errors
// and that the written size equals to the byte size
func TestAddFile(t *testing.T) {
	var assertion = assert.New(t)
	// create a new archive
	tmpArchiveFile, err := os.CreateTemp("", "*"+FileExt)
	assertion.NoError(err)

	archive := NewWriter(tmpArchiveFile)
	// run the test for the gzip implementation
	archivetest.AddFile(archive, t)
}

// TestAddFile tests if a file can be added to the archive without errors
// and that the written size equals to the byte size
func TestAddFileByPath(t *testing.T) {
	var assertion = assert.New(t)
	// create a new archive
	tmpArchiveFile, err := os.CreateTemp("", "*"+FileExt)
	assertion.NoError(err)

	archive := NewWriter(tmpArchiveFile)
	// run the test for the gzip implementation
	archivetest.AddFileByPath(archive, t)
}
