package std_session

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	watcherHttp "github.com/DaRealFreak/watcher-go/internal/http"
)

// trackingBody records whether Close was called so the test can assert the
// session released the (potentially slot-wrapped) response body.
type trackingBody struct {
	io.Reader
	closed bool
}

func (t *trackingBody) Close() error {
	t.closed = true
	return nil
}

// stubRoundTripper returns a fixed response regardless of the request.
type stubRoundTripper struct {
	resp *http.Response
}

func (s stubRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return s.resp, nil
}

// fatalErrorHandler flags any 500 response as a fatal (non-retryable) error.
type fatalErrorHandler struct{}

func (fatalErrorHandler) CheckResponse(response *http.Response) (error, bool) {
	if response.StatusCode == http.StatusInternalServerError {
		return fmt.Errorf("fatal status %d", response.StatusCode), true
	}
	return nil, false
}

func (fatalErrorHandler) CheckDownloadedFileForErrors(int64, http.Header) error { return nil }

func (fatalErrorHandler) IsFatalError(error) bool { return true }

// TestGet_FatalErrorClosesBody is a regression test: on a fatal error the
// session returns the response to the caller without retrying. Callers discard
// the response on error, so the session must close the body itself - otherwise
// the wrapped connection-budget slot leaks until the GC finalizer fires.
func TestGet_FatalErrorClosesBody(t *testing.T) {
	body := &trackingBody{Reader: strings.NewReader("boom")}
	s := NewStdClientSession("test")
	s.Client = &http.Client{Transport: stubRoundTripper{resp: &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       body,
		Header:     make(http.Header),
	}}}

	resp, err := s.Get("http://example.invalid", fatalErrorHandler{})
	if err == nil {
		t.Fatal("expected a fatal error, got nil")
	}
	if resp == nil {
		t.Fatal("expected the response to be returned alongside the fatal error")
	}
	if !body.closed {
		t.Fatal("response body was not closed on fatal error - budget slot leaks until GC")
	}
}

// compile-time check that the handler satisfies the interface used by Get.
var _ watcherHttp.StdClientErrorHandler = fatalErrorHandler{}
