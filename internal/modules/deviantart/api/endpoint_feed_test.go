package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeviantartAPI_FeedHomeBucket(t *testing.T) {
	daAPI.useConsoleExploit = false

	bucket, err := daAPI.FeedHomeBucket(BucketDeviationSubmitted, 0)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(bucket)

	// toggle console exploit, we also require the first OAuth2 process to have succeeded
	// since we require the user information cookie which is set on a successful login
	daAPI.useConsoleExploit = true

	bucketConsoleExploit, err := daAPI.FeedHomeBucket(BucketDeviationSubmitted, 0)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(bucketConsoleExploit)

	assert.New(t).Equal(bucket.Cursor, bucketConsoleExploit.Cursor)
}

func TestDeviantartAPI_FeedHome(t *testing.T) {
	daAPI.useConsoleExploit = false

	bucket, err := daAPI.FeedHomeBucket(BucketDeviationSubmitted, 0)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(bucket)

	continuation, err := daAPI.FeedHome(bucket.Cursor)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(continuation)

	// toggle console exploit, we also require the first OAuth2 process to have succeeded
	// since we require the user information cookie which is set on a successful login
	daAPI.useConsoleExploit = true

	bucketConsoleExploit, err := daAPI.FeedHomeBucket(BucketDeviationSubmitted, 0)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(bucketConsoleExploit)

	continuationConsoleExploit, err := daAPI.FeedHome(bucketConsoleExploit.Cursor)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(continuationConsoleExploit)

	assert.New(t).Equal(continuation.Cursor, continuationConsoleExploit.Cursor)
}
