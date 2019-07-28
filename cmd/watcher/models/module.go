package models

import (
	"net/url"
	"path/filepath"
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

// retrieve the file name of the passed uri
func (t *BaseModel) GetFileName(uri string) string {
	parsedUri, _ := url.Parse(uri)
	return filepath.Base(parsedUri.Path)
}

// retrieve the file extension of the passed uri
func (t *BaseModel) GetFileExtension(uri string) string {
	parsedUri, _ := url.Parse(uri)
	return filepath.Ext(parsedUri.Path)
}

// reverse the download queue items to get the oldest items first
// to be able to interrupt the update process anytime
func (t *BaseModel) ReverseDownloadQueueItems(downloadQueue []DownloadQueueItem) []DownloadQueueItem {
	for i, j := 0, len(downloadQueue)-1; i < j; i, j = i+1, j-1 {
		downloadQueue[i], downloadQueue[j] = downloadQueue[j], downloadQueue[i]
	}
	return downloadQueue
}
