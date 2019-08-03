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
func (session *Session) Get(uri string, tries int) (*http.Response, error) {
	// Get the data
	klog.Info(fmt.Sprintf("opening GET uri \"%s\" (try: %d)", uri, tries+1))
	response, err := session.Client.Get(uri)
	if err != nil {
		if session.MaxRetries >= tries {
			return nil, err
		} else {
			time.Sleep(time.Duration(tries+1) * time.Second)
			return session.Get(uri, tries+1)
		}
	}
	return response, err
}

// POST request
func (session *Session) Post(uri string, data url.Values, tries int) (*http.Response, error) {
	klog.Info(fmt.Sprintf("opening POST uri \"%s\" (try: %d)", uri, tries+1))
	response, err := session.Client.PostForm(uri, data)
	if err != nil {
		if session.MaxRetries >= tries {
			return nil, err
		} else {
			time.Sleep(time.Duration(tries+1) * time.Second)
			return session.Post(uri, data, tries+1)
		}
	}
	return response, err
}

// DownloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func (session *Session) DownloadFile(filepath string, uri string) error {
	klog.Info(fmt.Sprintf("downloading file: \"%s\" (uri: %s)", filepath, uri))
	// Get the data
	resp, err := session.Client.Get(uri)
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
	_, err = io.Copy(out, resp.Body)
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
