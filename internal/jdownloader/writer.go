package jdownloader

import (
	"net/url"
	"strings"
	"sync"
)

// Writer appends external links to a local ".crawljob" file and merges that
// file into JDownloader's Folder Watch directory. It is safe for concurrent
// use; the watcher may process items in parallel.
type Writer struct {
	mu  sync.Mutex //nolint:unused // used in concurrent write operations
	cfg Config
}

// NewWriter returns a Writer for the given config.
func NewWriter(cfg Config) *Writer {
	return &Writer{cfg: cfg}
}

// Enabled reports whether this writer's config has the handoff turned on.
// jdownloader.Default() builds its config from LoadConfig(), so for the default
// writer this reflects crawljob.enabled.
func (w *Writer) Enabled() bool {
	return w.cfg.Enabled
}

var (
	defaultWriter *Writer
	defaultOnce   sync.Once
)

// Default returns the process-wide Writer built from the global config.
func Default() *Writer {
	defaultOnce.Do(func() {
		defaultWriter = NewWriter(LoadConfig())
	})
	return defaultWriter
}

// Blacklisted reports whether rawURL's host equals or is a subdomain of any
// configured blacklist entry (case-insensitive).
func (w *Writer) Blacklisted(rawURL string) bool {
	host := hostOf(rawURL)
	if host == "" {
		return false
	}
	host = strings.ToLower(host)
	for _, d := range w.cfg.Blacklist {
		d = strings.ToLower(strings.TrimSpace(d))
		if d == "" {
			continue
		}
		if host == d || strings.HasSuffix(host, "."+d) {
			return true
		}
	}
	return false
}

// hostOf extracts the lowercase host from a URL, tolerating a missing scheme.
func hostOf(raw string) string {
	if raw == "" {
		return ""
	}
	if u, err := url.Parse(raw); err == nil && u.Hostname() != "" {
		return u.Hostname()
	}
	if u, err := url.Parse("https://" + raw); err == nil {
		return u.Hostname()
	}
	return ""
}
