package main

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// main function for testing
func TestRetrieveOAuth2Code(t *testing.T) {
	_ = assert.New(t)

	// ToDo: both cases:
	//  - calling http://lvh.me:8080/cb?code=test_code to retrieve "test_code"
	//  - timing out the function (60 seconds) to retrieve empty string
	log.Infof("deviantart oauth2 granted: %s", retrieveOAuth2Code())
}
