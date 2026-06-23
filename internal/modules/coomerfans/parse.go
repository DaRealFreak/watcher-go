// Package coomerfans contains the implementation of the coomerfans.com module
package coomerfans

import "regexp"

// postURLPattern matches a post URL: /p/{postId}/{userId}/{service}
var postURLPattern = regexp.MustCompile(`/p/(\d+)/(\d+)/(\w+)`)

// userURLPattern matches a creator URL: /u/{service}/{userId}/{username}
var userURLPattern = regexp.MustCompile(`/u/(\w+)/(\d+)/([^/?&]+)`)

// parsePostURL extracts the post ID, user ID and service from a post URL.
func parsePostURL(uri string) (postID, userID, service string, ok bool) {
	m := postURLPattern.FindStringSubmatch(uri)
	if len(m) != 4 {
		return "", "", "", false
	}
	return m[1], m[2], m[3], true
}

// parseUserURL extracts the service, user ID and username from a creator URL.
func parseUserURL(uri string) (service, userID, username string, ok bool) {
	m := userURLPattern.FindStringSubmatch(uri)
	if len(m) != 4 {
		return "", "", "", false
	}
	return m[1], m[2], m[3], true
}
