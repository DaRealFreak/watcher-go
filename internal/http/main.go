// Package http contains the basic HTTP functionality of the application
package http

import (
	"compress/gzip"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/PuerkitoBio/goquery"
	"github.com/spf13/viper"
)

// SessionInterface of used functions from the application to eventually change the underlying library
type SessionInterface interface {
	Get(uri string, errorHandlers ...ErrorHandler) (response *http.Response, err error)
	Post(uri string, data url.Values, errorHandlers ...ErrorHandler) (response *http.Response, err error)
	DownloadFile(filepath string, uri string, errorHandlers ...ErrorHandler) (err error)
	EnsureDownloadDirectory(fileName string)
	GetDocument(response *http.Response) *goquery.Document
	GetClient() *http.Client
	UpdateTreeFolderChangeTimes(filePath string)
	SetProxy(proxySettings *ProxySettings) (err error)
	SetClient(client *http.Client)
}

type ErrorHandler interface {
	CheckResponse(response *http.Response) (error error, fatal bool)
	CheckDownloadedFileForErrors(writtenSize int64, responseHeader http.Header) (err error)
}

// Session is an implementation to the SessionInterface to provide basic functions
type Session struct {
	SessionInterface
	ModuleKey string
}

// ProxySettings are the proxy server settings for the session
type ProxySettings struct {
	Enable   bool   `mapstructure:"enable"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Type     string `mapstructure:"type"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

// EnsureDownloadDirectory ensures that the download path already exists or creates it if not
// this function panics when path can't be created
func (s *Session) EnsureDownloadDirectory(fileName string) {
	dirName := filepath.Dir(fileName)
	if _, statError := os.Stat(dirName); statError != nil {
		mkdirError := os.MkdirAll(dirName, os.ModePerm)
		if mkdirError != nil {
			panic(mkdirError)
		}
	}
}

// GetDocument converts the http response to a *goquery.Document
func (s *Session) GetDocument(response *http.Response) *goquery.Document {
	var (
		reader io.ReadCloser
		err    error
	)

	switch response.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(response.Body)
		if err == nil {
			readerRes, readerErr := ioutil.ReadAll(reader)
			raven.CheckError(readerErr)

			response.Body = io.NopCloser(strings.NewReader(string(readerRes)))
		}
	}

	reader = response.Body

	defer raven.CheckClosure(reader)

	document, documentErr := goquery.NewDocumentFromReader(reader)
	raven.CheckError(documentErr)

	return document
}

// UpdateTreeFolderChangeTimes recursively updates the folder access and modification times
// to indicate changes in the data for file explorers
func (s *Session) UpdateTreeFolderChangeTimes(filePath string) {
	absFilePath, absErr := filepath.Abs(filePath)
	if absErr != nil {
		return
	}

	baseDirectory, baseDirErr := filepath.Abs(viper.GetString("download.directory"))
	if baseDirErr != nil {
		return
	}

	for {
		parentDir := filepath.Dir(absFilePath)
		// if we reached the top level or the module directory we break the update loop
		if parentDir == absFilePath || parentDir == baseDirectory {
			break
		}

		currentTime := time.Now().Local()
		if err := os.Chtimes(parentDir, currentTime, currentTime); err != nil {
			return
		}

		// update our file path for the parent folder
		absFilePath = parentDir
	}
}
