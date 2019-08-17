package session

import (
	"context"
	"fmt"
	watcherHttp "github.com/DaRealFreak/watcher-go/pkg/http"
	log "github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	"net/http"
	"net/http/cookiejar"
	"net/url"
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
		log.Info(fmt.Sprintf("downloading file: \"%s\" (uri: %s, try: %d)", filepath, uri, try))
		// ToDo: implement again
		// err = s.tryDownloadFile(filepath, uri)
		// if no error occurred return nil
		if err == nil {
			return
		} else {
			time.Sleep(time.Duration(try+1) * time.Second)
		}
	}
	return err
}

// wait for the leaky bucket to fill again
func (s *PixivSession) applyRateLimit() {
	// wait for request to stay within the rate limit
	if err := s.rateLimiter.Wait(s.ctx); err != nil {
		log.Fatal(err)
	}
}
