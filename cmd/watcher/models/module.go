package models

import (
	"regexp"
)

type ModuleInterface interface {
	Key() (key string)
	IsLoggedIn() (loggedIn bool)
	RegisterUriSchema(uriSchemas map[string][]*regexp.Regexp)
	Login(account *Account) (success bool)
	Parse(item *TrackedItem)
}

type DownloadQueueItem struct {
	ItemId      string
	DownloadTag string
	FileName    string
	FileUri     string
}

type Module struct {
	BaseModel
	ModuleInterface
}

type BaseModel struct {
}

func (t *BaseModel) GetFileName(uri string) string {
	// ToDo: implement getting the file name
	return uri
}

func (t *BaseModel) GetFileExtension(uri string) string {
	// ToDo: implement getting the file extension
	return uri
}

// reverse the download queue items to get the oldest items first
// to be able to interrupt the update process anytime
func (t *BaseModel) ReverseDownloadQueueItems(downloadQueue []DownloadQueueItem) []DownloadQueueItem {
	for i, j := 0, len(downloadQueue)-1; i < j; i, j = i+1, j-1 {
		downloadQueue[i], downloadQueue[j] = downloadQueue[j], downloadQueue[i]
	}
	return downloadQueue
}
