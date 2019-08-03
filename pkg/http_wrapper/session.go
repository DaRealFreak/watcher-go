package http_wrapper

import (
	"compress/gzip"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/kubernetes/klog"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Session struct {
	Client     *http.Client
	MaxRetries int
}

// initialize a new session and set all the required headers etc
func NewSession() *Session {
	jar, _ := cookiejar.New(nil)

	app := Session{
		Client:     &http.Client{Jar: jar},
		MaxRetries: 5,
	}
	return &app
}

// sends a GET request, returns the occurred error if something went wrong even after multiple tries
func (session *Session) Get(uri string) (response *http.Response, err error) {
	// access the passed url and return the data or the error which persisted multiple retries
	// post the request with the retries option
	for try := 1; try <= session.MaxRetries; try++ {
		klog.Info(fmt.Sprintf("opening GET uri \"%s\" (try: %d)", uri, try))
		response, err = session.Client.Get(uri)
		// if no error occurred break out of the loop
		if err == nil {
			break
		} else {
			time.Sleep(time.Duration(try+1) * time.Second)
		}
	}
	return response, err
}

// sends a POST request, returns the occurred error if something went wrong even after multiple tries
func (session *Session) Post(uri string, data url.Values) (response *http.Response, err error) {
	// post the request with the retries option
	for try := 1; try <= session.MaxRetries; try++ {
		klog.Info(fmt.Sprintf("opening POST uri \"%s\" (try: %d)", uri, try))
		response, err = session.Client.PostForm(uri, data)
		// if no error occurred break out of the loop
		if err == nil {
			break
		} else {
			time.Sleep(time.Duration(try+1) * time.Second)
		}
	}
	return response, err
}

// try to download the file, returns the occurred error if something went wrong even after multiple tries
func (session *Session) DownloadFile(filepath string, uri string) (err error) {
	for try := 1; try <= session.MaxRetries; try++ {
		klog.Info(fmt.Sprintf("downloading file: \"%s\" (uri: %s, try: %d)", filepath, uri, try))
		err = session.tryDownloadFile(filepath, uri)
		// if no error occurred return nil
		if err == nil {
			return
		} else {
			time.Sleep(time.Duration(try+1) * time.Second)
		}
	}
	return err
}

// this function will download a url to a local file.
// It's efficient because it will write as it downloads and not load the whole file into memory.
func (session *Session) tryDownloadFile(filepath string, uri string) error {
	// retrieve the data
	resp, err := session.Get(uri)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// ensure the directory
	session.ensureDownloadDirectory(filepath)

	// create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// write the body to file
	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	// additional validation to compare sent headers with the written file
	err = session.checkDownloadedFileForErrors(written, resp.Header)
	return err
}

// function to ensure that the download path already exists or create it if not
// this function panics when path can't be created
func (session *Session) ensureDownloadDirectory(fileName string) {
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
func (session *Session) checkDownloadedFileForErrors(writtenSize int64, responseHeader http.Header) (err error) {
	if val, ok := responseHeader["Content-Length"]; ok {
		fileSize, err := strconv.Atoi(val[0])
		if err == nil {
			if writtenSize != int64(fileSize) {
				err = fmt.Errorf("written file size doesn't match the header content length value")
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
