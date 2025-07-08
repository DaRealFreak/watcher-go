// Package http contains the basic HTTP functionality of the application
package http

import (
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/PuerkitoBio/goquery"
	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/spf13/viper"
	"golang.org/x/time/rate"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

// TlsClientSessionInterface of used functions from the application to eventually change the underlying library
type TlsClientSessionInterface interface {
	Get(uri string, errorHandlers ...TlsClientErrorHandler) (response *http.Response, err error)
	Post(uri string, data url.Values, errorHandlers ...TlsClientErrorHandler) (response *http.Response, err error)
	Do(req *http.Request, errorHandlers ...TlsClientErrorHandler) (response *http.Response, err error)
	DownloadFile(filepath string, uri string, errorHandlers ...TlsClientErrorHandler) (err error)
	DownloadFileFromResponse(response *http.Response, filepath string, errorHandlers ...TlsClientErrorHandler) (err error)
	EnsureDownloadDirectory(fileName string)
	GetDocument(response *http.Response) *goquery.Document
	GetClient() tls_client.HttpClient
	UpdateTreeFolderChangeTimes(filePath string)
	SetProxy(proxySettings *ProxySettings) (err error)
	SetClient(client tls_client.HttpClient)
	GetCookies(u *url.URL) []*http.Cookie
	SetCookies(u *url.URL, cookies []*http.Cookie)
	SetRateLimiter(rateLimiter *rate.Limiter)
}

type TlsClientErrorHandler interface {
	CheckResponse(response *http.Response) (error error, fatal bool)
	CheckDownloadedFileForErrors(writtenSize int64, responseHeader http.Header) (err error)
	IsFatalError(err error) bool
}

// TlsClientSession is an implementation to the TlsClientSessionInterface to provide basic functions
type TlsClientSession struct {
	TlsClientSessionInterface
	ModuleKey string
}

// Interface guard
var _ TlsClientSessionInterface = (*TlsClientSession)(nil)

// EnsureDownloadDirectory ensures that the download path already exists or creates it if not
// this function panics when path can't be created
func (s *TlsClientSession) EnsureDownloadDirectory(fileName string) {
	dirName := filepath.Dir(fileName)
	if _, statError := os.Stat(dirName); statError != nil {
		mkdirError := os.MkdirAll(dirName, os.ModePerm)
		if mkdirError != nil {
			panic(mkdirError)
		}
	}
}

// GetDocument converts the http response to a *goquery.Document
func (s *TlsClientSession) GetDocument(response *http.Response) *goquery.Document {
	defer raven.CheckClosure(response.Body)

	document, documentErr := goquery.NewDocumentFromReader(response.Body)
	raven.CheckError(documentErr)

	return document
}

// UpdateTreeFolderChangeTimes recursively updates the folder access and modification times
// to indicate changes in the data for file explorers
func (s *TlsClientSession) UpdateTreeFolderChangeTimes(filePath string) {
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
