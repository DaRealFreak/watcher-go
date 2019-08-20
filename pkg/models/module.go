package models

import (
	"fmt"
	"github.com/DaRealFreak/watcher-go/pkg/http"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

type ModuleInterface interface {
	Key() (key string)
	RequiresLogin() (requiresLogin bool)
	IsLoggedIn() bool
	RegisterURISchema(uriSchemas map[string][]*regexp.Regexp)
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
	Session  http.SessionInterface
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
	log.Info(fmt.Sprintf("found %d new items for uri: \"%s\"", len(downloadQueue), trackedItem.Uri))

	for index, data := range downloadQueue {
		log.Info(fmt.Sprintf("downloading updates for uri: \"%s\" (%0.2f%%)", trackedItem.Uri, float64(index+1)/float64(len(downloadQueue))*100))
		err := t.Session.DownloadFile(path.Join(viper.GetString("downloadDirectory"), t.Key(), data.DownloadTag, data.FileName), data.FileUri)
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		} else {
			t.DbIO.UpdateTrackedItem(trackedItem, data.ItemId)
		}
	}
}

// replaces reserved characters https://en.wikipedia.org/wiki/Filename#Reserved_characters_and_words
// and trims the result
func (t *Module) SanitizePath(path string, allowSeparator bool) string {
	var reservedCharacters *regexp.Regexp
	if allowSeparator {
		reservedCharacters = regexp.MustCompile("[:\"*?<>|]+")
	} else {
		reservedCharacters = regexp.MustCompile("[\\\\/:\"*?<>|]+")
	}
	path = reservedCharacters.ReplaceAllString(path, "_")
	for strings.Contains(path, "__") {
		path = strings.Replace(path, "__", "_", -1)
	}
	path = strings.Trim(path, "_")
	return path
}

// the default handling of errors in modules
func (t *Module) CheckError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
