package jdownloader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Writer appends external links to a local ".crawljob" file and merges that
// file into JDownloader's Folder Watch directory. It is safe for concurrent
// use; the watcher may process items in parallel.
type Writer struct {
	mu  sync.Mutex
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

// hostOf extracts the host from a URL, tolerating a missing scheme.
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

// Add appends one crawljob entry for the given links into the local file,
// skipping any link already present anywhere in the file. downloadFolder is
// resolved to an absolute path. If no links remain after dedupe, the file is
// left untouched.
func (w *Writer) Add(packageName, downloadFolder, sourceURL string, links []string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	abs, err := filepath.Abs(downloadFolder)
	if err != nil {
		return err
	}

	jobs, err := w.read()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	existing := make(map[string]bool)
	for _, j := range jobs {
		for _, l := range strings.Split(j.Text, "\n") {
			if l != "" {
				existing[l] = true
			}
		}
	}

	var fresh []string
	seen := make(map[string]bool)
	for _, l := range links {
		if l == "" || existing[l] || seen[l] {
			continue
		}
		seen[l] = true
		fresh = append(fresh, l)
	}
	if len(fresh) == 0 {
		return nil
	}

	jobs = append(jobs, Crawljob{
		PackageName:          packageName,
		DownloadFolder:       abs,
		Comment:              sourceURL,
		Text:                 strings.Join(fresh, "\n"),
		Enabled:              "true",
		AutoConfirm:          boolStatus(w.cfg.AutoConfirm),
		AutoStart:            boolStatus(w.cfg.AutoStart),
		ForcedStart:          "UNSET",
		ExtractAfterDownload: "UNSET",
	})

	return w.write(jobs)
}

// read loads the local crawljob file. A missing file returns an os.IsNotExist
// error; an empty file returns (nil, nil).
func (w *Writer) read() ([]Crawljob, error) {
	b, err := os.ReadFile(w.cfg.File)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(b)) == 0 {
		return nil, nil
	}
	var jobs []Crawljob
	if err := json.Unmarshal(b, &jobs); err != nil {
		return nil, fmt.Errorf("parse crawljob file %s: %w", w.cfg.File, err)
	}
	return jobs, nil
}

// write serializes jobs to the local crawljob file (pretty-printed so the user
// can edit it before merging).
func (w *Writer) write(jobs []Crawljob) error {
	if dir := filepath.Dir(w.cfg.File); dir != "" {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return err
		}
	}
	b, err := json.MarshalIndent(jobs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(w.cfg.File, b, 0o644)
}

// Queue routes a single external link to the crawljob when the handoff is
// enabled and the link's host is not blacklisted, logging the outcome under
// moduleKey (so the colored "module" log field still works). It returns true
// when it handled the link, so callers can `continue` instead of running their
// own fallback; it returns false when the handoff is disabled or the link is
// blacklisted, leaving the caller to fall back to its existing behavior.
func (w *Writer) Queue(moduleKey, packageName, downloadFolder, sourceURL, link string) bool {
	if !w.Enabled() || w.Blacklisted(link) {
		return false
	}
	if err := w.Add(packageName, downloadFolder, sourceURL, []string{link}); err != nil {
		slog.Warn(fmt.Sprintf("failed to add external URL %q to crawljob: %s", link, err.Error()), "module", moduleKey)
	} else {
		slog.Info(fmt.Sprintf("queued external URL %q for JDownloader (folder: %s)", link, downloadFolder), "module", moduleKey)
	}
	return true
}

// Merge moves the accumulated local crawljob file into JDownloader's Folder
// Watch directory under a unique name (watcher-go-<ts>.crawljob) so JDownloader
// ingests it fresh. Returns ("", nil) when there is nothing to merge. The
// caller supplies ts (e.g. time.Now().Unix()) so the operation stays
// deterministic and testable.
func (w *Writer) Merge(ts int64) (string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	jobs, err := w.read()
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	if len(jobs) == 0 {
		return "", nil
	}
	if w.cfg.FolderwatchPath == "" {
		return "", fmt.Errorf("crawljob.folderwatch_path is not configured")
	}
	if err := os.MkdirAll(w.cfg.FolderwatchPath, os.ModePerm); err != nil {
		return "", err
	}

	dest := filepath.Join(w.cfg.FolderwatchPath, fmt.Sprintf("watcher-go-%d.crawljob", ts))
	if err := moveFile(w.cfg.File, dest); err != nil {
		return "", err
	}
	return dest, nil
}

// moveFile renames src to dst, falling back to copy+remove across devices.
func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		_ = in.Close()
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = in.Close()
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		_ = in.Close()
		return err
	}
	if err := in.Close(); err != nil {
		return err
	}
	return os.Remove(src)
}
