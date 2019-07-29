package models

import (
	"fmt"
	"github.com/kubernetes/klog"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"watcher-go/cmd/watcher/arguments"
	"watcher-go/cmd/watcher/http_wrapper"
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
	ModuleInterface
	DbIO     DatabaseInterface
	Session  *http_wrapper.Session
	LoggedIn bool
}

// retrieve the file name of the passed uri
func (t *Module) GetFileName(uri string) string {
	parsedUri, _ := url.Parse(uri)
	return filepath.Base(parsedUri.Path)
}

// retrieve the file extension of the passed uri
func (t *Module) GetFileExtension(uri string) string {
	parsedUri, _ := url.Parse(uri)
	return filepath.Ext(parsedUri.Path)
}

// reverse the download queue items to get the oldest items first
// to be able to interrupt the update process anytime
func (t *Module) ReverseDownloadQueueItems(downloadQueue []DownloadQueueItem) []DownloadQueueItem {
	for i, j := 0, len(downloadQueue)-1; i < j; i, j = i+1, j-1 {
		downloadQueue[i], downloadQueue[j] = downloadQueue[j], downloadQueue[i]
	}
	return downloadQueue
}

func (t *Module) ProcessDownloadQueue(downloadQueue []DownloadQueueItem, trackedItem *TrackedItem) {
	// reverse queue to get the oldest "new" item first and manually update it
	downloadQueue = t.ReverseDownloadQueueItems(downloadQueue)
	klog.Info(fmt.Sprintf("found %d new items for uri: \"%s\"", len(downloadQueue), trackedItem.Uri))

	for index, data := range downloadQueue {
		klog.Info(fmt.Sprintf("downloading updates for uri: \"%s\" (%0.2f%%)", trackedItem.Uri, float64(index+1)/float64(len(downloadQueue))*100))
		_ = t.Session.DownloadFile(path.Join(*arguments.DownloadDirectory, t.Key(), data.DownloadTag, data.FileName), data.FileUri)
		t.DbIO.UpdateTrackedItem(trackedItem, data.ItemId)
	}
}
