// nolint: dupl
package zip

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/DaRealFreak/watcher-go/pkg/archive/archivetest"
	"github.com/stretchr/testify/assert"
)

// TestGetFiles tests if an archive can retrieve all file names/paths from the generated archive
func TestGetFiles(t *testing.T) {
	var assertion = assert.New(t)
	// create the archive
	tmpArchiveFile, err := ioutil.TempFile("", "*"+FileExt)
	assertion.NoError(err)

	writer := NewWriter(tmpArchiveFile)
	archivetest.GenerateTestFiles(t, writer)

	// open the archive and create a reader for it
	f, err := os.Open(tmpArchiveFile.Name())
	assertion.NoError(err)

	reader := NewReader(f)
	// archive
	archivetest.GetFiles(t, reader)
}

// TestGetFile tests if an archive can retrieve the passed file names from the archive
func TestGetFile(t *testing.T) {
	var assertion = assert.New(t)
	// create the archive
	tmpArchiveFile, err := ioutil.TempFile("", "*"+FileExt)
	assertion.NoError(err)

	writer := NewWriter(tmpArchiveFile)
	archivetest.GenerateTestFiles(t, writer)

	// open the archive and create a reader for it
	f, err := os.Open(tmpArchiveFile.Name())
	assertion.NoError(err)

	archiveReader := NewReader(f)
	// test if all files exist and the content is equal
	archivetest.GetFile(t, archiveReader)
}
