package session

import (
	"bytes"
	"context"
	"fmt"
	watcherHttp "github.com/DaRealFreak/watcher-go/pkg/http"
	log "github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"
)

type PixivSession struct {
	watcherHttp.Session
	HttpClient   *http.Client
	MobileClient *mobileClient
	rateLimiter  *rate.Limiter
	ctx          context.Context
	maxRetries   int
}

type mobileClient struct {
	OauthUrl     string
	headers      map[string]string
	ClientId     string
	ClientSecret string
	AccessToken  string
	RefreshToken string
}

// initialize a new session and set all the required headers etc
func NewSession() *PixivSession {
	jar, _ := cookiejar.New(nil)
	return &PixivSession{
		HttpClient: &http.Client{Jar: jar},
		MobileClient: &mobileClient{
			OauthUrl: "https://oauth.secure.pixiv.net/auth/token",
			headers: map[string]string{
				"App-OS":         "ios",
				"App-OS-Version": "10.3.1",
				"App-Version":    "6.7.1",
				"User-Agent":     "PixivIOSApp/6.7.1 (iOS 10.3.1; iPhone8,1)",
				"Referer":        "https://app-api.pixiv.net/",
				"Content-Type":   "application/x-www-form-urlencoded",
			},
			ClientId:     "MOBrBDS8blbauoSck0ZfDbtuzpyT",
			ClientSecret: "lsACyCD94FhDUtGTXi3QzcFE2uU1hqtDaKeqrdwj",
			AccessToken:  "",
			RefreshToken: "",
		},
		rateLimiter: rate.NewLimiter(rate.Every(1*time.Second), 1),
		ctx:         context.Background(),
		maxRetries:  5,
	}
}

// custom GET function to set headers like the mobile app
func (s *PixivSession) Get(uri string) (*http.Response, error) {
	s.applyRateLimit()

	log.Debug(fmt.Sprintf("opening GET uri %s (try: %d)", uri, 1))
	req, _ := http.NewRequest("GET", uri, nil)
	for headerKey, headerValue := range s.MobileClient.headers {
		req.Header.Add(headerKey, headerValue)
	}
	if s.MobileClient.AccessToken != "" {
		req.Header.Add("Authorization", "Bearer "+s.MobileClient.AccessToken)
	}
	res, err := s.HttpClient.Do(req)
	return res, err
}

// custom GET function to set headers like the mobile app
func (s *PixivSession) Post(uri string, data url.Values) (*http.Response, error) {
	s.applyRateLimit()

	log.Debug(fmt.Sprintf("opening POST uri %s (try: %d)", uri, 1))
	req, _ := http.NewRequest("POST", uri, strings.NewReader(data.Encode()))
	for headerKey, headerValue := range s.MobileClient.headers {
		req.Header.Add(headerKey, headerValue)
	}
	if s.MobileClient.AccessToken != "" {
		req.Header.Add("Authorization", "Bearer "+s.MobileClient.AccessToken)
	}
	res, err := s.HttpClient.Do(req)
	return res, err
}

// try to download the file, returns the occurred error if something went wrong even after multiple tries
func (s *PixivSession) DownloadFile(filepath string, uri string) (err error) {
	for try := 1; try <= s.maxRetries; try++ {
		log.Info(fmt.Sprintf("downloading file: %s (uri: %s, try: %d)", filepath, uri, try))
		err = s.tryDownloadFile(filepath, uri)
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
func (s *PixivSession) tryDownloadFile(filepath string, uri string) error {
	// retrieve the data
	resp, err := s.Get(uri)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	written, err := s.WriteToFile(filepath, content)

	// additional validation to compare sent headers with the written file
	err = s.CheckDownloadedFileForErrors(written, resp.Header)
	return err
}

// write content to file and return written amount of bytes and possible occurred errors
func (s *PixivSession) WriteToFile(filepath string, content []byte) (written int64, err error) {
	// ensure the directory
	s.EnsureDownloadDirectory(filepath)

	// create the file
	out, err := os.Create(filepath)
	if err != nil {
		return 0, err
	}
	defer out.Close()

	// write the body to file
	written, err = io.Copy(out, bytes.NewReader(content))
	if err != nil {
		return 0, err
	}
	return
}

// wait for the leaky bucket to fill again
func (s *PixivSession) applyRateLimit() {
	// wait for request to stay within the rate limit
	if err := s.rateLimiter.Wait(s.ctx); err != nil {
		log.Fatal(err)
	}
}
