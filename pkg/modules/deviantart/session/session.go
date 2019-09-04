package session

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/DaRealFreak/watcher-go/pkg/http/session"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	"github.com/EDDYCJY/fake-useragent"
	log "github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// DeviantArtSession is an extension of the DefaultSession, handling the OAuth2 Token process and POST requests
type DeviantArtSession struct {
	*session.DefaultSession
	TokenStore        *TokenStore
	UseConsoleExploit bool
}

// NewSession returns an initialized DeviantArtSession
func NewSession() *DeviantArtSession {
	ses := &DeviantArtSession{
		DefaultSession:    session.NewSession(),
		TokenStore:        NewTokenStore(),
		UseConsoleExploit: false,
	}
	ses.RateLimiter = rate.NewLimiter(rate.Every(5000*time.Millisecond), 1)
	return ses
}

// Post sends normal POST requests without any API scope/token contrary to the APIPost function
// values are handled similar to the http.Client PostForm function
func (s *DeviantArtSession) Post(uri string, data url.Values) (response *http.Response, err error) {
	return s.post(uri, data, "")
}

// post is the internal POST functionality which uses the handleRequest function
// to handle all error responses/expired tokens etc.
func (s *DeviantArtSession) post(uri string, data url.Values, scope string) (res *http.Response, err error) {
	return s.handleRequest(uri, data, scope,
		func(uri string, values url.Values) (res *http.Response, err error) {
			log.WithField("module", s.ModuleKey).Debugf("POST request: %s", uri)
			req, err := http.NewRequest("POST", uri, strings.NewReader(data.Encode()))
			if err != nil {
				log.Fatalln(err)
			}
			req.Header.Set("User-Agent", browser.Chrome())
			return s.Client.Do(req)
		},
	)
}

// APIPost handles the OAuth2 Token for the POST request including refresh for API requests
func (s *DeviantArtSession) APIPost(endpoint string, values url.Values, scopes ...string) (*http.Response, error) {
	// scopes are separated by whitespaces according to the docs
	// https://www.deviantart.com/developers/authentication
	scope := strings.Join(scopes, " ")

	if s.UseConsoleExploit {
		return s.handleRequest(endpoint, values, scope, s.APIConsoleExploit)
	}
	raven.CheckError(s.handleToken(&values, scope))
	// handle the request now as a normal POST request
	return s.post("https://www.deviantart.com/api/v1/oauth2"+endpoint, values, scope)
}

// APIGet handles the OAuth2 Token and scopes for the GET request including refresh for API requests
func (s *DeviantArtSession) APIGet(endpoint string, values url.Values, scopes ...string) (*http.Response, error) {
	// scopes are separated by whitespaces according to the docs
	// https://www.deviantart.com/developers/authentication
	scope := strings.Join(scopes, " ")

	if s.UseConsoleExploit {
		return s.handleRequest(endpoint, values, scope, s.APIConsoleExploit)
	}
	raven.CheckError(s.handleToken(&values, scope))
	return s.get("https://www.deviantart.com/api/v1/oauth2"+endpoint, values, scope)
}

// Get sends normal GET requests without any API scope/token contrary to the APIGet function
// values is also disabled for this response similar to the http.Client Get function
func (s *DeviantArtSession) Get(uri string) (res *http.Response, err error) {
	return s.get(uri, url.Values{}, "")
}

// get is the internal GET functionality which uses the handleRequest function
// to handle all error responses/expired tokens etc.
func (s *DeviantArtSession) get(uri string, values url.Values, scope string) (res *http.Response, err error) {
	return s.handleRequest(
		uri, values, scope,
		func(uri string, values url.Values) (res *http.Response, err error) {
			apiURL, err := url.Parse(uri)
			raven.CheckError(err)
			// parse existing fragments and override with passed values (required for token)
			fragments := apiURL.Query()
			for k, v := range values {
				fragments.Set(k, v[0])
			}
			apiURL.RawQuery = fragments.Encode()
			log.WithField("module", s.ModuleKey).Debugf("GET request: %s", apiURL.String())
			req, err := http.NewRequest("GET", apiURL.String(), nil)
			if err != nil {
				log.Fatalln(err)
			}
			req.Header.Set("User-Agent", browser.Chrome())
			return s.Client.Do(req)
		},
	)
}

// RefreshOAuth2Token tries to retrieve the OAuth2 Token from the API for passed scope, returns the success as boolean
func (s *DeviantArtSession) RefreshOAuth2Token(scope string) (success bool) {
	s.TokenStore.SetToken(scope, s.retrieveOAuth2Token(scope))
	return s.TokenStore.GetToken(scope) != nil
}

// APIConsoleExploit uses the developer console in the DeviantArt backend to use twice of the API rate limit
// and honestly, averaging 1 request every 180 seconds is horrendous for an API, even the now 1 request/90s is so low...
func (s *DeviantArtSession) APIConsoleExploit(endpoint string, values url.Values) (res *http.Response, err error) {
	values = url.Values{
		"c[]": {
			s.getDeveloperConsoleCommand(
				endpoint,
				values,
			),
		},
		"t": {"json"},
	}

	// retrieve the user info cookie for the ui argument (validated by the server)
	daURL, _ := url.Parse("https://deviantart.com")
	for _, cookie := range s.GetClient().Jar.Cookies(daURL) {
		if cookie.Name == "userinfo" {
			ui, _ := url.QueryUnescape(cookie.Value)
			values.Set("ui", ui)
			break
		}
	}

	log.WithField("module", s.ModuleKey).Debugf(
		"using developer console exploit for endpoint: %s", endpoint,
	)
	// send the POST request to the developer console
	res, err = s.post("https://www.deviantart.com/global/difi/?", values, "")
	if err != nil {
		return res, err
	}
	content, err := ioutil.ReadAll(res.Body)
	raven.CheckError(err)

	// unmarshal the response into the DeveloperConsoleResponse struct
	var consoleResponse DeveloperConsoleResponse
	raven.CheckError(json.Unmarshal(content, &consoleResponse))

	// if request was not successful we treat it as a 429 status code
	// we want to switch back the API mode to default and we most likely won't get any other errors anyway
	if consoleResponse.Response.Calls[0].Response.Status != "SUCCESS" {
		res.StatusCode = 429
	}

	marshalledResponse, err := json.Marshal(consoleResponse.DiFi.Response.Calls[0].Response.Content)
	raven.CheckError(err)
	// replace the response body with our marshalledResponse (equal to the direct OAuth2 application response)
	res.Body = ioutil.NopCloser(bytes.NewReader(marshalledResponse))
	return res, err
}

// getDeveloperConsoleCommand parses the endpoint and the values into the POST data for the development console request
func (s *DeviantArtSession) getDeveloperConsoleCommand(endpoint string, values url.Values) string {
	devConsoleCommand := "\"DeveloperConsole\",\"do_api_request\",[\"" + endpoint + "\",[%s]]"
	values.Set("da_version", "")
	values.Set("grant_type", "authorization_code")
	values.Set("mature_content", "true")
	values.Set("endpoint", endpoint)
	requestValues := make([]string, 0, len(values))
	for k, v := range values {
		requestValues = append(requestValues, "{\"name\":\""+k+"\",\"value\":\""+v[0]+"\"}")
	}
	return fmt.Sprintf(devConsoleCommand, strings.Join(requestValues, ","))
}

// handleRequest handles all requests and possible errors/expired tokens
// also switches the API mode if the limit got reached
func (s *DeviantArtSession) handleRequest(
	url string, values url.Values, scope string,
	getterFunc func(uri string, values url.Values) (res *http.Response, err error),
) (res *http.Response, err error) {
	switchMode := false
	// post the request with the retries option
	for try := 1; try <= s.MaxRetries; try++ {
		s.ApplyRateLimit()

		log.WithField("module", s.ModuleKey).Debug(
			fmt.Sprintf("opening URL %s (try: %d, scope: %s)", url, try, scope),
		)
		res, err = getterFunc(url, values)
		switch {
		case err == nil && res.StatusCode < 401:
			// normally everything < 400 is okay, but the API returns an error object on 400
			// and retrying won't help on that status code anyways
			s.checkAPIModeSwitch(switchMode)
			return res, err
		case err == nil && (res.StatusCode == 401 || res.StatusCode == 403) && scope != "":
			// on 401 or 403 we try to refresh our OAuth2 Token for the scope and try it again
			log.WithField("module", s.ModuleKey).Infof(
				"status code %d, refreshing OAuth2 Token", res.StatusCode,
			)
			if s.RefreshOAuth2Token(scope) {
				values.Set("access_token", s.TokenStore.GetToken(scope).AccessToken)
			}
		case res.StatusCode == 429:
			switchMode = true
			// the API limits are horrible, just sleep up to 5 minutes in which hopefully we get one more request in
			time.Sleep(time.Duration(try*20) * time.Second)
		default:
			// any other error falls into the retry clause
			time.Sleep(time.Duration(try*5) * time.Second)
		}
	}
	s.checkAPIModeSwitch(switchMode)
	return res, err
}

// checkAPIModeSwitch changes the API mode from OAuth2 application and development console if the change value is true
func (s *DeviantArtSession) checkAPIModeSwitch(change bool) {
	if change {
		log.WithField("module", s.ModuleKey).Debug("switching API mode to preserve limits")
		s.UseConsoleExploit = !s.UseConsoleExploit
	}
}

// handleToken checks if the token store has a value for the specified scope
// and overrides the access_token of the passed values if found
// retrieves the token from the API if unset so far
func (s *DeviantArtSession) handleToken(values *url.Values, scope string) error {
	// refresh OAuth2 Token if not set, return error if token couldn't get retrieved
	if !s.TokenStore.HasToken(scope) {
		if !s.RefreshOAuth2Token(scope) {
			return fmt.Errorf(
				"could not retrieve the OAuth2 Token from the API for scope %s",
				scope,
			)
		}
	}

	// add the access token if not already set
	if values.Get("access_token") == "" {
		values.Set("access_token", s.TokenStore.GetToken(scope).AccessToken)
	}
	return nil
}
