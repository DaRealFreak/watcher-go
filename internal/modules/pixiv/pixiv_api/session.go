package pixivapi

import (
	"crypto/md5"
	"fmt"
	http "github.com/bogdanfinn/fhttp"
	log "github.com/sirupsen/logrus"
	"net/url"
	"os"
	"strings"
	"time"
)

func (a *PixivAPI) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	return a.Do(req)
}

func (a *PixivAPI) Post(url string, data url.Values) (*http.Response, error) {
	formBody := data.Encode()
	req, err := http.NewRequest("POST", url, strings.NewReader(formBody))
	if err != nil {
		return nil, err
	}

	return a.Do(req)
}

func (a *PixivAPI) Do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Accept-Language", "en_US")
	req.Header.Set("App-OS", "ios")
	req.Header.Set("App-OS-Version", "14.6")
	req.Header.Set("App-Version", "5.0.156")
	req.Header.Set("Referer", a.referer)
	req.Header.Set("User-Agent", "PixivIOSApp/7.13.3 (iOS 14.6; iPhone13,2)")

	// add X-Client-Time and X-Client-Hash which are now getting validated server side
	localTime := time.Now()
	req.Header.Add("X-Client-Time", localTime.Format(time.RFC3339))
	req.Header.Add("X-Client-Hash", fmt.Sprintf(
		// nolint: gosec
		"%x", md5.Sum(
			[]byte(localTime.Format(time.RFC3339)+"28c1fdd170a5204386cb1313c7077b34f83e4aaf4aa829ce78c231e05b0bae2c"),
		),
	))

	// add Authorization header with the token
	if a.tokenSource != nil {
		token, tokenErr := a.tokenSource.Token()
		if tokenErr != nil {
			return nil, tokenErr
		}
		req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	}

	return a.Session.Do(req)
}

func (a *PixivAPI) DownloadFile(filepath string, uri string) (err error) {
	for try := 1; try <= 3; try++ {
		log.WithField("module", a.moduleKey).Debug(
			fmt.Sprintf("downloading file: \"%s\" (uri: %s, try: %d)", filepath, uri, try),
		)

		err = a.tryDownloadFile(filepath, uri)
		if err != nil {
			// sleep if an error occurred
			time.Sleep(time.Duration(try+1) * time.Second)
		} else {
			// if no error occurred return nil
			return nil
		}
	}

	if err != nil {
		// try to clean up failed file if it exists
		if _, statErr := os.Stat(filepath); statErr == nil {
			_ = os.Remove(filepath)
		}
	}

	return err
}

func (a *PixivAPI) tryDownloadFile(filepath string, uri string) error {
	// retrieve the data
	resp, err := a.Get(uri)
	if err != nil {
		return err
	}

	return a.Session.DownloadFileFromResponse(resp, filepath)
}
