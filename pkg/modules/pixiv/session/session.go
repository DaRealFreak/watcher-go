package session

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"

	watcherHttp "github.com/DaRealFreak/watcher-go/pkg/http"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	log "github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// PixivSession contains the implementation of the SessionInterface and custom required variables
type PixivSession struct {
	watcherHttp.Session
	Module       models.ModuleInterface
	HTTPClient   *http.Client
	MobileClient *mobileClient
	rateLimiter  *rate.Limiter
	ctx          context.Context
	MaxRetries   int
}

// mobileClient contains extracted variables of the official mobile application
type mobileClient struct {
	OauthURL     string
	headers      map[string]string
	ClientID     string
	ClientSecret string
	AccessToken  string
	RefreshToken string
}

// errorMessage is the JSON struct of API error messages
type errorMessage struct {
	UserMessage        string            `json:"user_message"`
	Message            string            `json:"message"`
	Reason             string            `json:"reason"`
	UserMessageDetails map[string]string `json:"user_message_details"`
}

// errorMessage is the JSON struct of API error responses
type errorResponse struct {
	Error *errorMessage `json:"error"`
}

// NewSession initializes a new session and sets all the required headers etc
func NewSession() *PixivSession {
	jar, _ := cookiejar.New(nil)
	return &PixivSession{
		HTTPClient: &http.Client{Jar: jar},
		MobileClient: &mobileClient{
			OauthURL: "https://oauth.secure.pixiv.net/auth/token",
			headers: map[string]string{
				"App-OS":         "ios",
				"App-OS-Version": "10.3.1",
				"App-Version":    "6.7.1",
				"User-Agent":     "PixivIOSApp/6.7.1 (iOS 10.3.1; iPhone8,1)",
				"Referer":        "https://app-api.pixiv.net/",
				"Content-Type":   "application/x-www-form-urlencoded",
			},
			ClientID:     "MOBrBDS8blbauoSck0ZfDbtuzpyT",
			ClientSecret: "lsACyCD94FhDUtGTXi3QzcFE2uU1hqtDaKeqrdwj",
			AccessToken:  "",
			RefreshToken: "",
		},
		rateLimiter: rate.NewLimiter(rate.Every(1*time.Second), 1),
		ctx:         context.Background(),
		MaxRetries:  5,
	}
}

// Get is a custom GET function to set headers like the mobile app
func (s *PixivSession) Get(uri string) (res *http.Response, err error) {
	// access the passed url and return the data or the error which persisted multiple retries
	// post the request with the retries option
	for try := 1; try <= s.MaxRetries; try++ {
		s.applyRateLimit()

		log.Debug(fmt.Sprintf("opening GET uri %s (try: %d)", uri, try))
		req, _ := http.NewRequest("GET", uri, nil)
		for headerKey, headerValue := range s.MobileClient.headers {
			req.Header.Add(headerKey, headerValue)
		}
		if s.MobileClient.AccessToken != "" {
			req.Header.Add("Authorization", "Bearer "+s.MobileClient.AccessToken)
		}
		res, err = s.HTTPClient.Do(req)
		if err == nil {
			bodyBytes, _ := ioutil.ReadAll(res.Body)
			if res != nil && res.StatusCode != 200 {
				raven.CheckError(errors.New(string(bodyBytes) + " (" + uri + ")"))
			}

			// check for API errors
			if s.containsAPIError(bodyBytes) {
				retry, err := s.handleAPIError(bodyBytes)
				if retry {
					return s.Get(uri)
				}
				return nil, err
			}

			// reset the response body to the original unread state
			res.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
			// break if we didn't reach any error state until the end of the loop
			break
		} else {
			// sleep if an error occurred
			time.Sleep(time.Duration(try+1) * time.Second)
		}
	}

	return res, err
}

// Post is a custom GET function to set headers like the mobile app
func (s *PixivSession) Post(uri string, data url.Values) (res *http.Response, err error) {
	// post the request with the retries option
	for try := 1; try <= s.MaxRetries; try++ {
		s.applyRateLimit()

		log.Debug(fmt.Sprintf("opening POST uri %s (try: %d)", uri, try))
		req, _ := http.NewRequest("POST", uri, strings.NewReader(data.Encode()))
		for headerKey, headerValue := range s.MobileClient.headers {
			req.Header.Add(headerKey, headerValue)
		}
		if s.MobileClient.AccessToken != "" {
			req.Header.Add("Authorization", "Bearer "+s.MobileClient.AccessToken)
		}
		res, err = s.HTTPClient.Do(req)
		if err == nil {
			bodyBytes, _ := ioutil.ReadAll(res.Body)
			if res != nil && res.StatusCode != 200 {
				raven.CheckError(errors.New(string(bodyBytes) + " (" + uri + ")"))
			}

			// check for API errors
			if s.containsAPIError(bodyBytes) {
				retry, err := s.handleAPIError(bodyBytes)
				if retry {
					return s.Post(uri, data)
				}
				return nil, err
			}

			// reset the response body to the original unread state
			res.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
			// break if we didn't reach any error state until the end of the loop
			break
		} else {
			// sleep if an error occurred
			time.Sleep(time.Duration(try+1) * time.Second)
		}
	}
	return res, err
}

// DownloadFile tries to download the file, returns the occurred error if something went wrong even after multiple tries
func (s *PixivSession) DownloadFile(filepath string, uri string) (err error) {
	for try := 1; try <= s.MaxRetries; try++ {
		log.Debug(fmt.Sprintf("downloading file: %s (uri: %s, try: %d)", filepath, uri, try))
		err = s.tryDownloadFile(filepath, uri)
		// if no error occurred return nil
		if err == nil {
			return
		}
		time.Sleep(time.Duration(try+1) * time.Second)
	}
	return err
}

// tryDownloadFile will download a url to a local file.
// It's efficient because it will write as it downloads and not load the whole file into memory.
func (s *PixivSession) tryDownloadFile(filepath string, uri string) error {
	// retrieve the data
	resp, err := s.Get(uri)
	if err != nil {
		return err
	}
	defer raven.CheckReadCloser(resp.Body)

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	written, err := s.WriteToFile(filepath, content)
	if err != nil {
		return err
	}

	// additional validation to compare sent headers with the written file
	err = s.CheckDownloadedFileForErrors(written, resp.Header)
	return err
}

// WriteToFile writes the passed content to file and returns the written amount of bytes and possible occurred errors
func (s *PixivSession) WriteToFile(filepath string, content []byte) (written int64, err error) {
	// ensure the directory
	s.EnsureDownloadDirectory(filepath)

	// create the file
	out, err := os.Create(filepath)
	if err != nil {
		return 0, err
	}
	defer raven.CheckReadCloser(out)

	// write the body to file
	written, err = io.Copy(out, bytes.NewReader(content))
	if err != nil {
		return 0, err
	}
	return
}

// applyRateLimit waits for the leaky bucket to fill again
func (s *PixivSession) applyRateLimit() {
	// wait for request to stay within the rate limit
	err := s.rateLimiter.Wait(s.ctx)
	raven.CheckError(err)
}

// containsAPIError checks if the returned value contains an error object
func (s *PixivSession) containsAPIError(response []byte) bool {
	var errorResponse errorResponse
	err := json.Unmarshal(response, &errorResponse)
	if err == nil && errorResponse.Error != nil && errorResponse.Error.Message != "" {
		return true
	}
	return false
}

// handleAPIError handles possible API errors and returns if the function should retry it
func (s *PixivSession) handleAPIError(response []byte) (retry bool, err error) {
	var errorResponse errorResponse
	_ = json.Unmarshal(response, &errorResponse)
	switch errorResponse.Error.Message {
	case "Error occurred at the OAuth process. " +
		"Please check your Access Token to fix this. " +
		"Error Message: invalid_grant":
		log.Info("access token expired, using refresh token to generate new token...")
		s.Module.Login(nil)
		return true, nil
	case "Rate Limit":
		log.Info("rate limit got exceeded, sleeping for 60 seconds...")
		time.Sleep(60 * time.Second)
		return true, nil
	}

	switch errorResponse.Error.UserMessage {
	case "該当作品は削除されたか、存在しない作品IDです。":
		return false, fmt.Errorf("requested art got removed or restricted")
	case "アクセスが制限されています。":
		return false, fmt.Errorf("requested user got restricted")
	}
	return
}
