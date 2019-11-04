// Package http contains the basic HTTP functionality of the application
package http

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/DaRealFreak/watcher-go/pkg/raven"
	"github.com/PuerkitoBio/goquery"
	"github.com/spf13/viper"
)

// SessionInterface of used functions from the application to eventually change the underlying library
type SessionInterface interface {
	Get(uri string) (response *http.Response, err error)
	Post(uri string, data url.Values) (response *http.Response, err error)
	DownloadFile(filepath string, uri string) (err error)
	EnsureDownloadDirectory(fileName string)
	CheckDownloadedFileForErrors(writtenSize int64, responseHeader http.Header) (err error)
	GetDocument(response *http.Response) *goquery.Document
	GetClient() *http.Client
	UpdateTreeFolderChangeTimes(filePath string)
	SetProxy(proxySettings *ProxySettings) (err error)
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

// CheckDownloadedFileForErrors compares the downloaded file with the content length header of the request if set
// also checks if the written bytes are more not equal or less than 0 which is definitely an unwanted result
func (s *Session) CheckDownloadedFileForErrors(writtenSize int64, responseHeader http.Header) (err error) {
	if val, ok := responseHeader["Content-Length"]; ok {
		fileSize, err := strconv.Atoi(val[0])
		if err == nil {
			if writtenSize != int64(fileSize) {
				return fmt.Errorf("written file size doesn't match the header content length value")
			}
		}
	}

	if writtenSize <= 0 {
		err = fmt.Errorf("written content has a size of 0 bytes")
	}

	return err
}

// GetDocument converts the http response to a *goquery.Document
func (s *Session) GetDocument(response *http.Response) *goquery.Document {
	var reader io.ReadCloser

	switch response.Header.Get("Content-Encoding") {
	case "gzip":
		reader, _ = gzip.NewReader(response.Body)
	default:
		reader = response.Body
	}

	defer raven.CheckClosure(reader)

	document, err := goquery.NewDocumentFromReader(reader)
	raven.CheckError(err)

	return document
}

// UpdateTreeFolderChangeTimes recursively updates the folder access and modification times
// to indicate changes in the data for file explorers
func (s *Session) UpdateTreeFolderChangeTimes(filePath string) {
	filePath, err := filepath.Abs(filePath)
	if err != nil {
		return
	}

	baseDirectory, err := filepath.Abs(viper.GetString("download.directory"))
	if err != nil {
		return
	}

	for {
		parentDir := filepath.Dir(filePath)
		// if we reached the top level or the module directory we break the update loop
		if parentDir == filePath || parentDir == baseDirectory {
			break
		}

		currentTime := time.Now().Local()

		err = os.Chtimes(parentDir, currentTime, currentTime)
		if err != nil {
			return
		}

		// update our file path for the parent folder
		filePath = parentDir
	}
}
