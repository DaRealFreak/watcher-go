package gdrive

import (
	"fmt"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"google.golang.org/api/drive/v3"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"
)

func (m *gdrive) parseFolder(item *models.TrackedItem) error {
	folderID := m.folderPattern.FindStringSubmatch(item.URI)[1]

	files, err := m.getFilesInFolder(folderID, "", 0)
	if err != nil {
		return err
	}

	sortedFiles := append(ByModifiedTime{}, files.Files...)
	sort.Sort(sortedFiles)

	for _, file := range sortedFiles {
		var res *http.Response

		for i := 0; i < 5; i++ {
			res, err = m.driveService.Files.Get(file.Id).Download()
			if err == nil {
				break
			}

			time.Sleep(time.Duration(i) * 5)
		}

		if err != nil {
			return err
		}

		m.Session.EnsureDownloadDirectory(file.Name)

		localFile, err := os.Create(file.Name)
		if err != nil {
			return err
		}

		_, err = io.Copy(localFile, res.Body)
		if err != nil {
			return err
		}

		fmt.Println(fmt.Sprintf("downloaded file: %s (id: %s)", file.Name, file.Id))
	}

	return nil
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
