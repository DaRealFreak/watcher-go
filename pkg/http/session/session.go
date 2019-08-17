package session

import (
	"fmt"
	watcherHttp "github.com/DaRealFreak/watcher-go/pkg/http"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"time"
)

type DefaultSession struct {
	watcherHttp.Session
	Client     *http.Client
	MaxRetries int
}

// initialize a new session and set all the required headers etc
func NewSession() *DefaultSession {
	jar, _ := cookiejar.New(nil)

	app := DefaultSession{
		Client:     &http.Client{Jar: jar},
		MaxRetries: 5,
	}
	return &app
}

// sends a GET request, returns the occurred error if something went wrong even after multiple tries
func (session *DefaultSession) Get(uri string) (response *http.Response, err error) {
	// access the passed url and return the data or the error which persisted multiple retries
	// post the request with the retries option
	for try := 1; try <= session.MaxRetries; try++ {
		log.Debug(fmt.Sprintf("opening GET uri \"%s\" (try: %d)", uri, try))
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
func (session *DefaultSession) Post(uri string, data url.Values) (response *http.Response, err error) {
	// post the request with the retries option
	for try := 1; try <= session.MaxRetries; try++ {
		log.Debug(fmt.Sprintf("opening POST uri \"%s\" (try: %d)", uri, try))
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
func (session *DefaultSession) DownloadFile(filepath string, uri string) (err error) {
	for try := 1; try <= session.MaxRetries; try++ {
		log.Info(fmt.Sprintf("downloading file: \"%s\" (uri: %s, try: %d)", filepath, uri, try))
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
func (session *DefaultSession) tryDownloadFile(filepath string, uri string) error {
	// retrieve the data
	resp, err := session.Get(uri)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// ensure the directory
	session.EnsureDownloadDirectory(filepath)

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
	err = session.CheckDownloadedFileForErrors(written, resp.Header)
	return err
}

// retrieve the client
func (session *DefaultSession) GetClient() *http.Client {
	return session.Client
}
