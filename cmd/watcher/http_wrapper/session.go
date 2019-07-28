package http_wrapper

import (
	"github.com/PuerkitoBio/goquery"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
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
func (session *Session) Get(url string, tries int) (*http.Response, error) {
	// Get the data
	response, err := session.Client.Get(url)
	if err != nil {
		if session.MaxRetries >= tries {
			return nil, err
		} else {
			time.Sleep(time.Duration(tries+1) * time.Second)
			return session.Get(url, tries+1)
		}
	}
	return response, err
}

// POST request
func (session *Session) Post(uri string, data url.Values, tries int) (*http.Response, error) {
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
func (session *Session) DownloadFile(filepath string, url string) error {
	// Get the data
	resp, err := session.Client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

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

// convert the http response to a goquery document
func (session *Session) GetDocument(response *http.Response) *goquery.Document {
	defer response.Body.Close()
	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		log.Fatal("Error loading HTTP response body. ", err)
	}
	return document
}