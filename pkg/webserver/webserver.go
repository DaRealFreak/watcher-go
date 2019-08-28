package webserver

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/mitchellh/go-homedir"

	"github.com/DaRealFreak/watcher-go/pkg/raven"
	log "github.com/sirupsen/logrus"
)

// nolint: gochecknoglobals
var (
	Server         = newWebServer()
	LocalDomainTLD = "lvh.me"
)

// WebServer contains all relevant information for managing the local web server
type WebServer struct {
	Mux         *http.ServeMux
	Srv         *http.Server
	currentJobs uint
	ctx         context.Context
}

// newWebServer initializes the web server and registers a handler
func newWebServer() *WebServer {
	mux := http.NewServeMux()
	return &WebServer{
		Mux: mux,
		Srv: &http.Server{
			Addr:    ":8080",
			Handler: mux,
		},
		currentJobs: 0,
		ctx:         context.Background(),
	}
}

// StartWebServer starts the local web server if not already running and increases the current job count by one
func StartWebServer() {
	// increase the jobs we are waiting for
	Server.currentJobs++
	// ignore if server is already running
	if Server.currentJobs > 1 {
		log.Debug("local web server is already running")
		return
	}
	// listen with a go routine to be able to time it out
	go func() {
		// retrieve the home dir to place certificates in our default folder
		home, err := homedir.Dir()
		raven.CheckError(err)

		// certificate and key pem
		certificateDir := filepath.Join(home, "/.watcher/certificates/")
		localCert := filepath.Join(certificateDir, LocalDomainTLD+".pem")
		localKeyFile := filepath.Join(certificateDir, LocalDomainTLD+".key.pem")
		raven.CheckError(ensureCertificates(localCert, localKeyFile))

		// listen and serve with the local certificate
		// returns ErrServerClosed on graceful close
		if err := Server.Srv.ListenAndServeTLS(
			localCert,
			localKeyFile,
		); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %s", err)
		}
	}()
}

// StopWebServer stops the local web server if no more jobs are open to handle
// also resets the server handler, so after restarting the handlers would have to be registered again
func StopWebServer() {
	// check if we have at least one running job
	if Server.currentJobs == 0 {
		log.Warning("no jobs are currently running")
		return
	}
	// mark one job as done
	Server.currentJobs--
	// if we still have jobs to do, don't shut down the server
	if Server.currentJobs > 0 {
		log.Debugf("local web server still has %d jobs to do", Server.currentJobs)
		return
	}
	log.Info("shutting down local web server")
	raven.CheckError(Server.Srv.Shutdown(Server.ctx))
	resetServerHandler()
}

// ForceStopWebServer forces the local web server to shut down regardless of jobs
// useful on quitting the application
func ForceStopWebServer() {
	if Server.currentJobs > 0 {
		log.Info("force shutting down local web server")
		raven.CheckError(Server.Srv.Shutdown(Server.ctx))
		resetServerHandler()
		Server.currentJobs = 0
	}
}

// resetServerHandler resets the server handler to be able to register a handler function on the same pattern again
func resetServerHandler() {
	log.Debug("resetting server handler")
	Server.Mux = http.NewServeMux()
	Server.Srv.Handler = Server.Mux
}

func ensureCertificates(certFile string, keyFile string) (err error) {
	// ensure the directory for the files
	for _, fileName := range []string{certFile, keyFile} {
		dirName := filepath.Dir(fileName)
		if _, statError := os.Stat(dirName); statError != nil {
			if err := os.MkdirAll(dirName, os.ModePerm); err != nil {
				return err
			}
		}
	}

	// if either of the files does not exist we run the mkcert command and return the value
	for _, fileName := range []string{certFile, keyFile} {
		if _, err := os.Stat(fileName); os.IsNotExist(err) {
			return exec.Command("mkcert", "-cert-file", certFile, "-key-file", keyFile, LocalDomainTLD).Run()
		}
	}
	return nil
}
