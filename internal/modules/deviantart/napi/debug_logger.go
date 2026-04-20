package napi

import (
	"bytes"
	"fmt"
	http "github.com/bogdanfinn/fhttp"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// RequestLogger dumps every request/response that flows through napi.do()
// into sequenced files under the configured output directory.
type RequestLogger struct {
	Dir     string
	counter uint64
	mu      sync.Mutex
}

// NewRequestLogger creates a logger that writes to dir (created if needed).
func NewRequestLogger(dir string) *RequestLogger {
	_ = os.MkdirAll(dir, 0755)
	return &RequestLogger{Dir: dir}
}

func (l *RequestLogger) nextIndex() uint64 {
	return atomic.AddUint64(&l.counter, 1) - 1
}

func timeTag() string {
	return time.Now().Format("150405") // HHMMSS
}

// sanitizeEndpoint turns a URL path like /_puppy/dadeviation/init into _puppy_dadeviation_init
func sanitizeEndpoint(u *url.URL) string {
	p := strings.ReplaceAll(u.Path, "/", "_")
	if len(p) > 80 {
		p = p[:80]
	}
	return p
}

// LogRequest writes the request file.
func (l *RequestLogger) LogRequest(req *http.Request) {
	idx := l.nextIndex()
	ts := timeTag()
	endpoint := sanitizeEndpoint(req.URL)

	name := fmt.Sprintf("%05d_%s_%s_%s_req.txt", idx, ts, req.Method, endpoint)

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%s %s\n", req.Method, req.URL.String())
	fmt.Fprintf(&buf, "Time: %s\n", time.Now().Format(time.RFC3339))
	buf.WriteString("\n=== Headers ===\n")

	keys := make([]string, 0, len(req.Header))
	for k := range req.Header {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&buf, "%s: %s\n", k, strings.Join(req.Header[k], "; "))
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	_ = os.WriteFile(filepath.Join(l.Dir, name), buf.Bytes(), 0644)
}

// LogDownloadRequest writes a request log for a file download (wixmp CDN etc).
func (l *RequestLogger) LogDownloadRequest(method string, uri string) {
	idx := l.nextIndex()
	ts := timeTag()

	parsed, _ := url.Parse(uri)
	endpoint := "download"
	if parsed != nil {
		endpoint = sanitizeEndpoint(parsed)
	}

	name := fmt.Sprintf("%05d_%s_%s_%s_req.txt", idx, ts, method, endpoint)

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%s %s\n", method, uri)
	fmt.Fprintf(&buf, "Time: %s\n", time.Now().Format(time.RFC3339))
	buf.WriteString("\n=== Download Request (via TlsClientSession) ===\n")

	l.mu.Lock()
	defer l.mu.Unlock()
	_ = os.WriteFile(filepath.Join(l.Dir, name), buf.Bytes(), 0644)
}

// LogDownloadResponse writes response meta for a download attempt.
// For error responses (status >= 400) it also captures the body.
func (l *RequestLogger) LogDownloadResponse(uri string, statusCode int, statusText string, headers http.Header, body []byte) {
	idx := l.nextIndex()
	ts := timeTag()

	parsed, _ := url.Parse(uri)
	endpoint := "download"
	if parsed != nil {
		endpoint = sanitizeEndpoint(parsed)
	}

	metaName := fmt.Sprintf("%05d_%s_%s_download_resp_meta.txt", idx, ts, endpoint)

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Status: %d %s\n", statusCode, statusText)
	fmt.Fprintf(&buf, "URL: %s\n", uri)
	fmt.Fprintf(&buf, "Time: %s\n", time.Now().Format(time.RFC3339))

	if headers != nil {
		buf.WriteString("\n=== Response Headers ===\n")
		keys := make([]string, 0, len(headers))
		for k := range headers {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(&buf, "%s: %s\n", k, strings.Join(headers[k], "; "))
		}
	}

	buf.WriteString("\n")

	l.mu.Lock()
	defer l.mu.Unlock()
	_ = os.WriteFile(filepath.Join(l.Dir, metaName), buf.Bytes(), 0644)

	// Also dump body for error responses
	if statusCode >= 400 && len(body) > 0 {
		bodyName := fmt.Sprintf("%05d_%s_%s_download_resp_body.txt", idx, ts, endpoint)
		_ = os.WriteFile(filepath.Join(l.Dir, bodyName), body, 0644)
	}
}

// LogResponse writes the response meta and body files.
// It reads and re-buffers the body so the caller can still consume it.
func (l *RequestLogger) LogResponse(req *http.Request, resp *http.Response, csrfToken string, jar func(u *url.URL) []*http.Cookie) {
	idx := l.nextIndex()
	ts := timeTag()
	endpoint := sanitizeEndpoint(req.URL)

	// --- response body ---
	body, _ := io.ReadAll(resp.Body)
	resp.Body = io.NopCloser(bytes.NewReader(body))

	bodyName := fmt.Sprintf("%05d_%s_%s_resp_body.json", idx, ts, endpoint)
	_ = os.WriteFile(filepath.Join(l.Dir, bodyName), body, 0644)

	// --- response meta ---
	metaName := fmt.Sprintf("%05d_%s_%s_resp_meta.txt", idx, ts, endpoint)

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Status: %d %s\n", resp.StatusCode, resp.Status)
	fmt.Fprintf(&buf, "URL: %s\n", req.URL.String())
	fmt.Fprintf(&buf, "Time: %s\n", time.Now().Format(time.RFC3339))

	buf.WriteString("\n=== Response Headers ===\n")
	keys := make([]string, 0, len(resp.Header))
	for k := range resp.Header {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&buf, "%s: %s\n", k, strings.Join(resp.Header[k], "; "))
	}

	fmt.Fprintf(&buf, "\n=== Current NAPI CSRFToken ===\n%s\n", csrfToken)

	if jar != nil {
		buf.WriteString("\n=== Cookies In Jar (after response) ===\n")
		for _, c := range jar(req.URL) {
			fmt.Fprintf(&buf, "%s = %s\n", c.Name, c.Value)
		}
	}

	buf.WriteString("\n")

	l.mu.Lock()
	defer l.mu.Unlock()
	_ = os.WriteFile(filepath.Join(l.Dir, metaName), buf.Bytes(), 0644)
}
