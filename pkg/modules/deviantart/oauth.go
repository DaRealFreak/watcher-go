package main

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/DaRealFreak/watcher-go/pkg/raven"
	log "github.com/sirupsen/logrus"
)

// WebServerTimeout default timeout for web server to retrieve OAuth2 token
const WebServerTimeout = 60 * time.Second

// DeviantArtAPIClientID is the client ID of the registered application accessing the API
// can be retrieved from https://www.deviantart.com/developers/apps
const DeviantArtAPIClientID = "9991"

// oAuth2 is used to retrieve the OAuth2 code
type oAuth2 struct {
	code    string
	granted chan bool
}

// NewOAuth2Application creates the granted channel
func NewOAuth2Application() *oAuth2 {
	return &oAuth2{
		granted: make(chan bool),
	}
}

// oAuth2ApplicationCallback handles web requests and passes it to the granted channel
// if a granted query fragment is present
func (a *oAuth2) oAuth2ApplicationCallback(w http.ResponseWriter, r *http.Request) {
	q, _ := url.ParseQuery(r.URL.RawQuery)
	if len(q["code"]) > 0 {
		a.code = q["code"][0]
		a.granted <- true
	}
	log.Debug("request received from", r.RemoteAddr)
}

// retrieveOAuth2Code creates a new OAuth2Application, starts a local web server on port 8080
// and waits for a request containing the OAuth2 token for the API
// server will shut down after 60 seconds automatically and return an empty string
func retrieveOAuth2Code() string {
	oAuth2Application := NewOAuth2Application()
	srv := &http.Server{Addr: ":8080"}
	http.HandleFunc("/cb", oAuth2Application.oAuth2ApplicationCallback)
	log.Debug("starting local web server at port 8080")

	// listen with a go routine to be able to time it out
	go func() {
		// returns ErrServerClosed on graceful close
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %s", err)
		}
	}()
	// print DeviantArt URL to retrieve the grant granted
	log.Infof("open following url to retrieve the granted: %s",
		"https://www.deviantart.com/oauth2/authorize?response_type=code"+
			"&client_id="+DeviantArtAPIClientID+"&redirect_uri=http://lvh.me:8080/cb&scope=basic&state=mysessionid",
	)

	select {
	case <-oAuth2Application.granted:
		log.Info("callback with granted received, shutting down local web server")
		raven.CheckError(srv.Shutdown(context.Background()))
		return oAuth2Application.code
	case <-time.After(WebServerTimeout):
		log.Warningf("no callback with granted received within %d seconds, shutting down local web server",
			int(WebServerTimeout.Seconds()),
		)
		return ""
	}
}
