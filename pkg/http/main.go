package http

import (
	"compress/gzip"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
)

type SessionInterface interface {
	Get(uri string) (response *http.Response, err error)
	Post(uri string, data url.Values) (response *http.Response, err error)
	DownloadFile(filepath string, uri string) (err error)
	EnsureDownloadDirectory(fileName string)
	CheckDownloadedFileForErrors(writtenSize int64, responseHeader http.Header) (err error)
	GetDocument(response *http.Response) *goquery.Document
	GetClient() *http.Client
}

type Session struct {
	SessionInterface
}

// function to ensure that the download path already exists or create it if not
// this function panics when path can't be created
func (session *Session) EnsureDownloadDirectory(fileName string) {
	dirName := filepath.Dir(fileName)
	if _, statError := os.Stat(dirName); statError != nil {
		mkdirError := os.MkdirAll(dirName, os.ModePerm)
		if mkdirError != nil {
			panic(mkdirError)
		}
	}
}

// compare the downloaded file with the content length header of the request if set
// also checks if the written bytes are more not equal or less than 0 which is definitely an unwanted result
func (session *Session) CheckDownloadedFileForErrors(writtenSize int64, responseHeader http.Header) (err error) {
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

// convert the http response to a goquery document
func (session *Session) GetDocument(response *http.Response) *goquery.Document {
	var reader io.ReadCloser
	switch response.Header.Get("Content-Encoding") {
	case "gzip":
		reader, _ = gzip.NewReader(response.Body)
		defer reader.Close()
	default:
		reader = response.Body
		defer response.Body.Close()
	}
	document, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		log.Fatal("Error loading HTTP response body. ", err)
	}
	return document
}
