package http

import (
	"compress/gzip"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/PuerkitoBio/goquery"
	"github.com/spf13/viper"
	"golang.org/x/time/rate"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// StdClientSessionInterface of used functions from the application to eventually change the underlying library
type StdClientSessionInterface interface {
	Get(uri string, errorHandlers ...StdClientErrorHandler) (response *http.Response, err error)
	Post(uri string, data url.Values, errorHandlers ...StdClientErrorHandler) (response *http.Response, err error)
	Do(req *http.Request, errorHandlers ...StdClientErrorHandler) (response *http.Response, err error)
	DownloadFile(filepath string, uri string, errorHandlers ...StdClientErrorHandler) (err error)
	DownloadFileFromResponse(response *http.Response, filepath string, errorHandlers ...StdClientErrorHandler) (err error)
	EnsureDownloadDirectory(fileName string)
	GetDocument(response *http.Response) *goquery.Document
	GetClient() *http.Client
	UpdateTreeFolderChangeTimes(filePath string)
	SetProxy(proxySettings *ProxySettings) (err error)
	SetClient(client *http.Client)
	GetCookies(u *url.URL) []*http.Cookie
	SetCookies(u *url.URL, cookies []*http.Cookie)
	SetRateLimiter(rateLimiter *rate.Limiter)
}

type StdClientErrorHandler interface {
	CheckResponse(response *http.Response) (error error, fatal bool)
	CheckDownloadedFileForErrors(writtenSize int64, responseHeader http.Header) (err error)
	IsFatalError(err error) bool
}

// StdClientSession is an implementation to the StdClientSessionInterface to provide basic functions
type StdClientSession struct {
	StdClientSessionInterface
	ModuleKey string
}

// Interface guard
var _ StdClientSessionInterface = (*StdClientSession)(nil)

// EnsureDownloadDirectory ensures that the download path already exists or creates it if not
// this function panics when path can't be created
func (s *StdClientSession) EnsureDownloadDirectory(fileName string) {
	dirName := filepath.Dir(fileName)
	if _, statError := os.Stat(dirName); statError != nil {
		mkdirError := os.MkdirAll(dirName, os.ModePerm)
		if mkdirError != nil {
			panic(mkdirError)
		}
	}
}

// GetDocument converts the http response to a *goquery.Document
func (s *StdClientSession) GetDocument(response *http.Response) *goquery.Document {
	var (
		reader io.ReadCloser
		err    error
	)

	switch response.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(response.Body)
		if err == nil {
			readerRes, readerErr := io.ReadAll(reader)
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
func (s *StdClientSession) UpdateTreeFolderChangeTimes(filePath string) {
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
