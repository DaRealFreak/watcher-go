package fourchan

import (
	"errors"
	"io"
	"net/url"
	"strings"
	"testing"

	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
)

// fakeClient is a minimal tls_client.HttpClient used to drive the "does the original
// thread still exist" probe without a network request. Only Get is implemented; any
// other method panics via the embedded nil interface to surface unexpected usage.
type fakeClient struct {
	tls_client.HttpClient
	resp  *fhttp.Response
	err   error
	calls int
}

func (f *fakeClient) Get(string) (*fhttp.Response, error) {
	f.calls++

	return f.resp, f.err
}

// closeTrackingBody records whether the probe closed the response body.
type closeTrackingBody struct {
	io.Reader
	closed bool
}

func (c *closeTrackingBody) Close() error {
	c.closed = true

	return nil
}

// newThreadTestModule builds a 4chan module whose session hands out the passed client
// for the thread existence probe.
func newThreadTestModule(client *fakeClient) *fourChan {
	m := NewBareModule().ModuleInterface.(*fourChan)
	m.Session = &fakeSession{client: client}

	return m
}

const testChanURL = "https://boards.4chan.org/d/thread/123456"

// TestThreadRemovedIgnoresTransportErrors is a regression test: the probe uses the raw
// client, which returns a *nil* response next to a transport error (e.g. the fatal
// "tls: invalid server key share"). Reading StatusCode off it panicked. A failed probe
// must also not report the thread as removed, since that would complete a tracked item
// because of a temporary network problem.
func TestThreadRemovedIgnoresTransportErrors(t *testing.T) {
	client := &fakeClient{err: &url.Error{
		Op:  "Get",
		URL: testChanURL,
		Err: errors.New("tls: invalid server key share"),
	}}
	m := newThreadTestModule(client)

	if m.threadRemoved(testChanURL) {
		t.Error("a failed probe must not report the thread as removed")
	}
	if client.calls != 1 {
		t.Errorf("expected exactly one probe request, got %d", client.calls)
	}
}

// TestThreadRemovedOnNotFound covers the actual purpose of the probe: a 404 means the
// original thread is gone, so the item may be marked complete.
func TestThreadRemovedOnNotFound(t *testing.T) {
	body := &closeTrackingBody{Reader: strings.NewReader("not found")}
	client := &fakeClient{resp: &fhttp.Response{StatusCode: fhttp.StatusNotFound, Body: body}}
	m := newThreadTestModule(client)

	if !m.threadRemoved(testChanURL) {
		t.Error("a 404 response must report the thread as removed")
	}
	if !body.closed {
		t.Error("probe response body was not closed")
	}
}

// TestThreadRemovedOnExistingThread ensures a live thread is never completed.
func TestThreadRemovedOnExistingThread(t *testing.T) {
	body := &closeTrackingBody{Reader: strings.NewReader("<html></html>")}
	client := &fakeClient{resp: &fhttp.Response{StatusCode: fhttp.StatusOK, Body: body}}
	m := newThreadTestModule(client)

	if m.threadRemoved(testChanURL) {
		t.Error("a 200 response must not report the thread as removed")
	}
	if !body.closed {
		t.Error("probe response body was not closed")
	}
}

// TestThreadRemovedHandlesMissingBody guards the close path against a response without
// a body, which the raw client can return for HEAD-like or error paths.
func TestThreadRemovedHandlesMissingBody(t *testing.T) {
	client := &fakeClient{resp: &fhttp.Response{StatusCode: fhttp.StatusNotFound}}
	m := newThreadTestModule(client)

	if !m.threadRemoved(testChanURL) {
		t.Error("a 404 without a body must still report the thread as removed")
	}
}
