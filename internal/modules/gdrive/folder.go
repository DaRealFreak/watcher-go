package gdrive

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"google.golang.org/api/drive/v3"
)

func (m *gdrive) parseFolder(item *models.TrackedItem) error {
	folderID := m.folderPattern.FindStringSubmatch(item.URI)[1]
	lastModifiedTimestamp, _ := strconv.ParseInt(item.CurrentItem, 10, 64)
	lastModified := time.Unix(0, lastModifiedTimestamp)

	files, err := m.getFilesInFolder(folderID, "", 0)
	if err != nil {
		return err
	}

	sortedFiles := append(ByModifiedTime{}, files.Files...)
	sort.Sort(sortedFiles)

	for i, file := range sortedFiles {
		modified, _ := time.Parse(time.RFC3339Nano, file.ModifiedTime)
		if modified.After(lastModified) {
			// all files after the current one are new too since it's already sorted by modified date
			sortedFiles = sortedFiles[i:]
			break
		}

		if i == len(sortedFiles)-1 {
			// all files are older than our last update, so return here
			return nil
		}
	}

	return m.downloadFiles(sortedFiles, item)
}

func (m *gdrive) getFilesInFolder(folderID string, base string, depth uint) (*drive.FileList, error) {
	folder, err := m.driveService.Files.Get(folderID).Do()
	if err != nil {
		return nil, err
	}

	base = filepath.Join(base, folder.Name)

	list, err := m.driveService.Files.List().
		OrderBy("modifiedTime desc").
		Q(fmt.Sprintf("'%s' in parents", folderID)).
		Fields("*").
		Do()
	if err != nil {
		return nil, err
	}

	// iterate in reverse so we can remove the folders from the file list using the iteration index
	for i := len(list.Files) - 1; i >= 0; i-- {
		file := list.Files[i]
		if file.MimeType == "application/vnd.google-apps.folder" {
			files, err := m.getFilesInFolder(file.Id, base, depth+1)
			if err != nil {
				return nil, err
			}

			list.Files = append(list.Files[:i], list.Files[i+1:]...)
			list.Files = append(list.Files, files.Files...)
		} else {
			file.Name = filepath.Join(base, file.Name)
		}
	}

	return list, nil
}
