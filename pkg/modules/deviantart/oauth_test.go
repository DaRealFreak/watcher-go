package main

import (
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRetrieveOAuth2Code tests the local http server to retrieve the OAuth2 code
func TestRetrieveOAuth2Code(t *testing.T) {
	assertion := assert.New(t)

	var wg sync.WaitGroup
	wg.Add(1)
	// routine this check so we can actually call the request
	go checkNoTimeout(t, &wg)
	// call local host resolved domain with test_code as code
	_, err := http.Get("http://lvh.me:8080/cb?code=test_code")
	assertion.NoError(err)
	// wait for this test to finish
	wg.Wait()

	// wait for timeout returning empty string
	oAuth2Code := retrieveOAuth2Code()
	assertion.Equal(oAuth2Code, "")
}

// checkNoTimeout checks the web server response if actually called
// to test this we have to goroutine this check and lock it to prevent overlapping port bindings
func checkNoTimeout(t *testing.T, wg *sync.WaitGroup) {
	assertion := assert.New(t)
	ch := make(chan string)
	// listen with a go routine to be able to time it out
	go func() {
		ch <- retrieveOAuth2Code()
	}()

	receivedCode := <-ch
	assertion.Equal(receivedCode, "test_code")
	wg.Done()
}
