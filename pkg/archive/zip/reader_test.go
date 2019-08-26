package zip

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetFiles tests if an archive can retrieve all file names/paths from the generated archive
func TestGetFiles(t *testing.T) {
	var assertion = assert.New(t)
	files := map[string]string{
		"accounts.sql":        "INSERT INTO test; COMMIT;",
		"README":              "some meaningful content",
		"content_dir/LICENSE": "license content",
		"empty_dir/":          "",
	}

	tmpArchive := generateTestFile(t, files)
	f, err := os.Open(tmpArchive)
	assertion.NoError(err)
	archiveReader := NewReader(f)
	archiveFiles, err := archiveReader.GetFiles()
	assertion.NoError(err)
	assertion.Equal(len(files), len(archiveFiles))
}

// generateTestFile generates a new archive with the passed files
func generateTestFile(t *testing.T, files map[string]string) string {
	var assertion = assert.New(t)
	// create a new archive
	tmpArchiveFile, err := ioutil.TempFile("", "*"+FileExt)
	assertion.NoError(err)
	archive := NewArchiveWriter(tmpArchiveFile)

	for fileName, fileContent := range files {
		_, err = archive.AddFile(fileName, []byte(fileContent))
		assertion.NoError(err)
	}
	err = archive.Close()
	assertion.NoError(err)
	return tmpArchiveFile.Name()
}
