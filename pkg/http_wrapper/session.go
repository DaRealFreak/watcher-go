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

// GET request
func (session *Session) Get(uri string) (*http.Response, error) {
	// access the passed url and return the data or the error which persisted multiple retries
	return session.getRetries(uri, 0)
}

func (session *Session) getRetries(uri string, tries int) (*http.Response, error) {
	klog.Info(fmt.Sprintf("opening GET uri \"%s\" (try: %d)", uri, tries+1))
	response, err := session.Client.Get(uri)
	if err != nil {
		if session.MaxRetries >= tries {
			return nil, err
		} else {
			time.Sleep(time.Duration(tries+1) * time.Second)
			return session.getRetries(uri, tries+1)
		}
	}
	return response, err
}

// POST request
func (session *Session) Post(uri string, data url.Values) (*http.Response, error) {
	// post the request with the retries option
	return session.postRetries(uri, data, 0)
}

func (session *Session) postRetries(uri string, data url.Values, tries int) (*http.Response, error) {
	klog.Info(fmt.Sprintf("opening POST uri \"%s\" (try: %d)", uri, tries+1))
	response, err := session.Client.PostForm(uri, data)
	if err != nil {
		if session.MaxRetries >= tries {
			return nil, err
		} else {
			time.Sleep(time.Duration(tries+1) * time.Second)
			return session.postRetries(uri, data, tries+1)
		}
	}
	return response, err
}

// DownloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func (session *Session) DownloadFile(filepath string, uri string) error {
	klog.Info(fmt.Sprintf("downloading file: \"%s\" (uri: %s)", filepath, uri))
	// Get the data
	resp, err := session.Get(uri)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// ensure the directory
	session.ensureDownloadDirectory(filepath)

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	err = session.checkDownloadedFileForErrors(written, resp.Header)
	return err
}

func (session *Session) ensureDownloadDirectory(fileName string) {
	dirName := filepath.Dir(fileName)
	if _, statError := os.Stat(dirName); statError != nil {
		mkdirError := os.MkdirAll(dirName, os.ModePerm)
		if mkdirError != nil {
			panic(mkdirError)
		}
	}
}

func (session *Session) checkDownloadedFileForErrors(writtenSize int64, responseHeader http.Header) (err error) {
	if val, ok := responseHeader["Content-Length"]; ok {
		fileSize, err := strconv.Atoi(val[0])
		if err == nil {
			if writtenSize != int64(fileSize) {
				err = fmt.Errorf("written file size doesn't match the header content length value")
			}
			fmt.Println(writtenSize, fileSize)
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
