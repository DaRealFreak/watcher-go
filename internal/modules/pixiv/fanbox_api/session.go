package fanboxapi

import (
	"fmt"
	http "github.com/bogdanfinn/fhttp"
	log "github.com/sirupsen/logrus"
	"os"
	"time"
)

func (a *FanboxAPI) get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	return a.do(req)
}

func (a *FanboxAPI) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Referer", "https://www.fanbox.cc")
	req.Header.Set("Origin", "https://www.fanbox.cc")
	req.Header.Set("User-Agent", a.UserAgent)

	// set the required cookies for the request
	cookieString := fmt.Sprintf("FANBOXSESSID=%s", a.SessionCookie.Value)
	if a.CfClearanceCookie != nil {
		cookieString += fmt.Sprintf("; %s=%s", a.CfClearanceCookie.Name, a.CfClearanceCookie.Value)
	}
	req.Header.Set("Cookie", cookieString)

	return a.Session.Do(req)
}

func (a *FanboxAPI) DownloadFile(filepath string, uri string) (err error) {
	for try := 1; try <= 3; try++ {
		log.WithField("module", a.Key).Debug(
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

func (a *FanboxAPI) tryDownloadFile(filepath string, uri string) error {
	// retrieve the data
	resp, err := a.get(uri)
	if err != nil {
		return err
	}

	return a.Session.DownloadFileFromResponse(resp, filepath)
}
