# JDownloader `.crawljob` Handoff Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Accumulate external download links that watcher-go can't download itself (mega, mediafire, …) into a JDownloader `.crawljob` file with correct per-post destination folders, and add a `watcher crawljob merge` command that pushes that file into JDownloader's Folder Watch directory.

**Architecture:** A new shared package `internal/jdownloader/` owns a concurrency-safe `Writer` (config, blacklist, append-with-dedupe, merge) plus a `Crawljob` storable. The four post-based modules (pawchive, kemono, patreon, pixiv) route their unparseable external links into the shared writer at the point they currently warn-and-drop. A new Cobra command flushes the file.

**Tech Stack:** Go (stdlib `encoding/json`, `net/url`, `os`, `path/filepath`, `sync`), `spf13/viper` (config), `spf13/cobra` (CLI). No new third-party dependencies.

## Global Constraints

- **Go module:** `github.com/DaRealFreak/watcher-go`. Import the new package as `github.com/DaRealFreak/watcher-go/internal/jdownloader`.
- **Do NOT create commits automatically.** Per the user's git workflow rule, every task ends at a verified, commit-ready state; the **user** stages and commits. Where a task says "commit," that means *the work is ready for the user to commit* — do not run `git commit` yourself.
- **Commit convention (for when the user commits):** `[TASK] description` — lowercase imperative, no period.
- **Lint is a required gate:** `golangci-lint run` (golangci-lint v2, config `.golangci.yml`) must pass with zero errors before a task is considered done.
- **JDownloader `.crawljob` field encoding:** `enabled` is the JSON string `"true"`. The `BooleanStatus` fields (`autoConfirm`, `autoStart`, `forcedStart`, `extractAfterDownload`) use the string enum `"TRUE"` / `"FALSE"` / `"UNSET"`.
- **`downloadFolder` must be an absolute path** — JDownloader requires it; watcher-go's download dir may be relative.
- **Concurrency:** the watcher may process items in parallel (`Run.RunParallel`), so all `Writer` file operations must be mutex-guarded.
- **Routing rule:** only links `factory.CanParse` returns **false** for go to the crawljob. Parseable links keep their existing native-download behavior.
- **No behavior change when crawljob is disabled:** with `crawljob.enabled` unset/false, every module must behave exactly as it does today.
- **Tests:** plain `testing` package (no testify), matching the existing style in `internal/modules/pawchive/download_test.go`.

---

## File Structure

- `internal/jdownloader/crawljob.go` — `Crawljob` storable struct + `boolStatus` helper.
- `internal/jdownloader/config.go` — `Config` struct, `LoadConfig()`, `Enabled()`.
- `internal/jdownloader/writer.go` — `Writer`, `NewWriter`, `Default`, `Blacklisted`, `Add`, `Merge`, file IO helpers.
- `internal/jdownloader/*_test.go` — unit tests for the above.
- `cmd/watcher/crawljob.go` — `watcher crawljob merge` command.
- `cmd/watcher/main.go` — register the new command (1 line).
- `internal/modules/pawchive/download.go` — route unparseable links to the writer + gate change.
- `internal/modules/kemono/download.go` — same.
- `internal/modules/patreon/download.go` — same (also add a `CanParse` guard it currently lacks).
- `internal/modules/pixiv/download.go` — route unparseable caption links (pixiv only prints today).

---

### Task 1: jdownloader package — `Crawljob` storable and config

**Files:**
- Create: `internal/jdownloader/crawljob.go`
- Create: `internal/jdownloader/config.go`
- Test: `internal/jdownloader/crawljob_test.go`
- Test: `internal/jdownloader/config_test.go`

**Interfaces:**
- Produces:
  - `type Crawljob struct { ... }` — see fields below.
  - `func boolStatus(b bool) string` — returns `"TRUE"`/`"FALSE"`.
  - `type Config struct { Enabled bool; File string; FolderwatchPath string; Blacklist []string; AutoStart bool; AutoConfirm bool }` with mapstructure tags.
  - `func LoadConfig() Config` — reads viper key `crawljob` with defaults.
  - `func Enabled() bool` — `viper.GetBool("crawljob.enabled")`.

- [ ] **Step 1: Write the failing test for `Crawljob` JSON encoding**

Create `internal/jdownloader/crawljob_test.go`:

```go
package jdownloader

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCrawljobJSONEncoding(t *testing.T) {
	job := Crawljob{
		PackageName:          "pawchive.st - 123 - Title",
		DownloadFolder:       `C:\Downloads\x`,
		Comment:              "https://pawchive.st/p/123",
		Text:                 "https://mega.nz/file/a\nhttps://mediafire.com/b",
		Enabled:              "true",
		AutoConfirm:          boolStatus(true),
		AutoStart:            boolStatus(false),
		ForcedStart:          "UNSET",
		ExtractAfterDownload: "UNSET",
	}

	b, err := json.Marshal([]Crawljob{job})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)

	for _, want := range []string{
		`"packageName":"pawchive.st - 123 - Title"`,
		`"downloadFolder":"C:\\Downloads\\x"`,
		`"enabled":"true"`,
		`"autoConfirm":"TRUE"`,
		`"autoStart":"FALSE"`,
		`"forcedStart":"UNSET"`,
		`"extractAfterDownload":"UNSET"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("encoded crawljob missing %q\ngot: %s", want, s)
		}
	}
}

func TestBoolStatus(t *testing.T) {
	if boolStatus(true) != "TRUE" {
		t.Errorf("boolStatus(true) = %q", boolStatus(true))
	}
	if boolStatus(false) != "FALSE" {
		t.Errorf("boolStatus(false) = %q", boolStatus(false))
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/jdownloader/... -run 'TestCrawljobJSONEncoding|TestBoolStatus' -v`
Expected: FAIL — package/identifiers `Crawljob`, `boolStatus` undefined (build error).

- [ ] **Step 3: Implement `crawljob.go`**

Create `internal/jdownloader/crawljob.go`:

```go
// Package jdownloader writes JDownloader Folder Watch ".crawljob" files for
// external links that watcher-go cannot download itself, so JDownloader can
// fetch them straight into the correct per-post folder.
package jdownloader

// Crawljob is one entry of a JDownloader Folder Watch ".crawljob" file
// (a JSON array of these). JDownloader's FolderWatch extension ingests the
// file and adds each package.
//
// The BooleanStatus fields (AutoConfirm/AutoStart/ForcedStart/
// ExtractAfterDownload) use the string enum "TRUE"/"FALSE"/"UNSET". Enabled
// is the JSON string "true".
type Crawljob struct {
	PackageName          string `json:"packageName"`
	DownloadFolder       string `json:"downloadFolder"`
	Comment              string `json:"comment,omitempty"`
	Text                 string `json:"text"`
	Enabled              string `json:"enabled"`
	AutoConfirm          string `json:"autoConfirm"`
	AutoStart            string `json:"autoStart"`
	ForcedStart          string `json:"forcedStart"`
	ExtractAfterDownload string `json:"extractAfterDownload"`
}

// boolStatus maps a Go bool to JDownloader's BooleanStatus string enum.
func boolStatus(b bool) string {
	if b {
		return "TRUE"
	}
	return "FALSE"
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/jdownloader/... -run 'TestCrawljobJSONEncoding|TestBoolStatus' -v`
Expected: PASS.

- [ ] **Step 5: Write the failing test for config defaults**

Create `internal/jdownloader/config_test.go`:

```go
package jdownloader

import (
	"testing"

	"github.com/spf13/viper"
)

func TestLoadConfigDefaults(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	cfg := LoadConfig()
	if cfg.Enabled {
		t.Errorf("Enabled should default to false")
	}
	if cfg.File != "./watcher-go.crawljob" {
		t.Errorf("File default = %q, want ./watcher-go.crawljob", cfg.File)
	}
	if !cfg.AutoStart {
		t.Errorf("AutoStart should default to true")
	}
	if !cfg.AutoConfirm {
		t.Errorf("AutoConfirm should default to true")
	}
}

func TestLoadConfigOverrides(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	viper.Set("crawljob.enabled", true)
	viper.Set("crawljob.file", "/tmp/x.crawljob")
	viper.Set("crawljob.folderwatch_path", "/jd/folderwatch")
	viper.Set("crawljob.auto_start", false)
	viper.Set("crawljob.blacklist", []string{"discord.gg", "patreon.com"})

	cfg := LoadConfig()
	if !cfg.Enabled {
		t.Errorf("Enabled override not applied")
	}
	if cfg.File != "/tmp/x.crawljob" {
		t.Errorf("File = %q", cfg.File)
	}
	if cfg.FolderwatchPath != "/jd/folderwatch" {
		t.Errorf("FolderwatchPath = %q", cfg.FolderwatchPath)
	}
	if cfg.AutoStart {
		t.Errorf("AutoStart override (false) not applied")
	}
	if cfg.AutoConfirm != true {
		t.Errorf("AutoConfirm should still default true when unset")
	}
	if len(cfg.Blacklist) != 2 || cfg.Blacklist[0] != "discord.gg" {
		t.Errorf("Blacklist = %v", cfg.Blacklist)
	}
	if !Enabled() {
		t.Errorf("Enabled() should report true")
	}
}
```

- [ ] **Step 6: Run the test to verify it fails**

Run: `go test ./internal/jdownloader/... -run 'TestLoadConfig' -v`
Expected: FAIL — `LoadConfig`, `Config`, `Enabled` undefined.

- [ ] **Step 7: Implement `config.go`**

Create `internal/jdownloader/config.go`:

```go
package jdownloader

import "github.com/spf13/viper"

// defaultFile is the local accumulation file used when crawljob.file is unset.
const defaultFile = "./watcher-go.crawljob"

// Config holds the global "crawljob" settings block.
type Config struct {
	Enabled         bool     `mapstructure:"enabled"`
	File            string   `mapstructure:"file"`
	FolderwatchPath string   `mapstructure:"folderwatch_path"`
	Blacklist       []string `mapstructure:"blacklist"`
	AutoStart       bool     `mapstructure:"auto_start"`
	AutoConfirm     bool     `mapstructure:"auto_confirm"`
}

// LoadConfig reads the global "crawljob" config from viper. AutoStart and
// AutoConfirm default to true and File to defaultFile; mapstructure only
// overwrites fields that are actually present in the config, so unset booleans
// keep these defaults.
func LoadConfig() Config {
	cfg := Config{
		File:        defaultFile,
		AutoStart:   true,
		AutoConfirm: true,
	}
	_ = viper.UnmarshalKey("crawljob", &cfg)
	if cfg.File == "" {
		cfg.File = defaultFile
	}
	return cfg
}

// Enabled reports whether the crawljob handoff is turned on. It reads viper
// directly so module gate checks don't need to build a Writer.
func Enabled() bool {
	return viper.GetBool("crawljob.enabled")
}
```

- [ ] **Step 8: Run the tests to verify they pass**

Run: `go test ./internal/jdownloader/... -v`
Expected: PASS (all four tests).

- [ ] **Step 9: Lint and build**

Run: `golangci-lint run ./internal/jdownloader/...` then `go build -v .`
Expected: no lint errors, build succeeds. Task is now commit-ready (user commits).

---

### Task 2: Writer — construction, `Enabled`, `Blacklisted`

**Files:**
- Create: `internal/jdownloader/writer.go`
- Test: `internal/jdownloader/writer_test.go`

**Interfaces:**
- Consumes: `Config` (Task 1).
- Produces:
  - `type Writer struct { ... }` (holds `cfg Config` and a `sync.Mutex`).
  - `func NewWriter(cfg Config) *Writer`.
  - `func Default() *Writer` — process-wide singleton built from `LoadConfig()`.
  - `func (w *Writer) Blacklisted(rawURL string) bool` — case-insensitive domain-suffix match.

- [ ] **Step 1: Write the failing test**

Create `internal/jdownloader/writer_test.go`:

```go
package jdownloader

import "testing"

func TestBlacklisted(t *testing.T) {
	w := NewWriter(Config{Blacklist: []string{"mediafire.com", "Discord.gg", "t.me"}})

	cases := map[string]bool{
		"https://www.mediafire.com/file/abc":  true,  // subdomain match
		"https://mediafire.com/file/abc":      true,  // exact host match
		"http://cdn.discord.gg/x":             true,  // case-insensitive blacklist entry
		"https://t.me/somechannel":            true,
		"https://mega.nz/file/abc":            false, // not listed
		"https://notmediafire.com/file/abc":   false, // must not match by substring
		"mega.nz/file/abc":                    false, // scheme-less, not listed
		"":                                    false,
	}
	for raw, want := range cases {
		if got := w.Blacklisted(raw); got != want {
			t.Errorf("Blacklisted(%q) = %v, want %v", raw, got, want)
		}
	}
}

func TestBlacklistedSchemeless(t *testing.T) {
	w := NewWriter(Config{Blacklist: []string{"mediafire.com"}})
	if !w.Blacklisted("www.mediafire.com/file/abc") {
		t.Errorf("scheme-less blacklisted host should still match")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/jdownloader/... -run 'Blacklisted' -v`
Expected: FAIL — `NewWriter` / `Writer` undefined.

- [ ] **Step 3: Implement the Writer skeleton + `Blacklisted`**

Create `internal/jdownloader/writer.go`:

```go
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
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/jdownloader/... -run 'Blacklisted' -v`
Expected: PASS.

- [ ] **Step 5: Lint and build**

Run: `golangci-lint run ./internal/jdownloader/...` then `go build -v .`
Expected: clean. Commit-ready.

---

### Task 3: Writer — `Add` (append with dedupe, absolute folder)

**Files:**
- Modify: `internal/jdownloader/writer.go`
- Test: `internal/jdownloader/writer_test.go`

**Interfaces:**
- Produces:
  - `func (w *Writer) Add(packageName, downloadFolder, sourceURL string, links []string) error`
  - Internal helpers `func (w *Writer) read() ([]Crawljob, error)` and `func (w *Writer) write(jobs []Crawljob) error`.

- [ ] **Step 1: Write the failing tests**

Append to `internal/jdownloader/writer_test.go`:

```go
import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func readJobs(t *testing.T, path string) []Crawljob {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var jobs []Crawljob
	if err := json.Unmarshal(b, &jobs); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return jobs
}

func TestAddWritesEntry(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "out.crawljob")
	w := NewWriter(Config{File: file, AutoStart: true, AutoConfirm: true})

	if err := w.Add("pkg1", dir, "https://src/1", []string{"https://mega.nz/a", "https://mega.nz/b"}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	jobs := readJobs(t, file)
	if len(jobs) != 1 {
		t.Fatalf("want 1 job, got %d", len(jobs))
	}
	if jobs[0].Text != "https://mega.nz/a\nhttps://mega.nz/b" {
		t.Errorf("Text = %q", jobs[0].Text)
	}
	if jobs[0].AutoStart != "TRUE" || jobs[0].AutoConfirm != "TRUE" {
		t.Errorf("auto flags = %q/%q", jobs[0].AutoStart, jobs[0].AutoConfirm)
	}
	if jobs[0].Enabled != "true" {
		t.Errorf("Enabled = %q", jobs[0].Enabled)
	}
	if !filepath.IsAbs(jobs[0].DownloadFolder) {
		t.Errorf("DownloadFolder must be absolute, got %q", jobs[0].DownloadFolder)
	}
}

func TestAddResolvesRelativeFolder(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "out.crawljob")
	w := NewWriter(Config{File: file})

	if err := w.Add("pkg", "relative/sub", "https://src", []string{"https://mega.nz/a"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	jobs := readJobs(t, file)
	if !filepath.IsAbs(jobs[0].DownloadFolder) {
		t.Errorf("relative folder not made absolute: %q", jobs[0].DownloadFolder)
	}
}

func TestAddDedupesAcrossEntries(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "out.crawljob")
	w := NewWriter(Config{File: file})

	if err := w.Add("pkg1", dir, "https://src/1", []string{"https://mega.nz/a"}); err != nil {
		t.Fatalf("Add 1: %v", err)
	}
	// second call: one duplicate, one new
	if err := w.Add("pkg2", dir, "https://src/2", []string{"https://mega.nz/a", "https://mega.nz/c"}); err != nil {
		t.Fatalf("Add 2: %v", err)
	}

	jobs := readJobs(t, file)
	if len(jobs) != 2 {
		t.Fatalf("want 2 jobs, got %d", len(jobs))
	}
	if jobs[1].Text != "https://mega.nz/c" {
		t.Errorf("second entry should only contain the new link, got %q", jobs[1].Text)
	}
}

func TestAddAllDuplicatesIsNoop(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "out.crawljob")
	w := NewWriter(Config{File: file})

	_ = w.Add("pkg1", dir, "https://src/1", []string{"https://mega.nz/a"})
	if err := w.Add("pkg2", dir, "https://src/2", []string{"https://mega.nz/a"}); err != nil {
		t.Fatalf("Add 2: %v", err)
	}
	jobs := readJobs(t, file)
	if len(jobs) != 1 {
		t.Errorf("all-duplicate Add must not append, got %d jobs", len(jobs))
	}
}

func TestQueueHandledWhenEnabled(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "out.crawljob")
	w := NewWriter(Config{Enabled: true, File: file})

	if handled := w.Queue("mod", "pkg", dir, "https://src", "https://mega.nz/a"); !handled {
		t.Fatalf("Queue should report handled when enabled and not blacklisted")
	}
	if jobs := readJobs(t, file); len(jobs) != 1 || jobs[0].Text != "https://mega.nz/a" {
		t.Errorf("Queue did not add the link: %+v", jobs)
	}
}

func TestQueueNotHandledWhenDisabled(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "out.crawljob")
	w := NewWriter(Config{Enabled: false, File: file})

	if handled := w.Queue("mod", "pkg", dir, "https://src", "https://mega.nz/a"); handled {
		t.Errorf("Queue should report not-handled when disabled")
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Errorf("disabled Queue must not create the file")
	}
}

func TestQueueNotHandledWhenBlacklisted(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "out.crawljob")
	w := NewWriter(Config{Enabled: true, File: file, Blacklist: []string{"mega.nz"}})

	if handled := w.Queue("mod", "pkg", dir, "https://src", "https://mega.nz/a"); handled {
		t.Errorf("Queue should report not-handled when blacklisted (caller falls back)")
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Errorf("blacklisted Queue must not create the file")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/jdownloader/... -run 'TestAdd|TestQueue' -v`
Expected: FAIL — `Add` / `Queue` undefined.

- [ ] **Step 3: Implement `Add`, `read`, `write`**

Append to `internal/jdownloader/writer.go` (add `bytes`, `encoding/json`, `fmt`, `log/slog`, `os`, `path/filepath` to the import block — `log/slog` is used by `Queue` in the next step):

```go
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
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/jdownloader/... -run 'TestAdd|TestQueue' -v`
Expected: PASS (all seven).

- [ ] **Step 5: Lint and build**

Run: `golangci-lint run ./internal/jdownloader/...` then `go build -v .`
Expected: clean. Commit-ready.

---

### Task 4: Writer — `Merge` (flush into folderwatch)

**Files:**
- Modify: `internal/jdownloader/writer.go`
- Test: `internal/jdownloader/writer_test.go`

**Interfaces:**
- Produces: `func (w *Writer) Merge(ts int64) (movedTo string, err error)` — moves the local file into `FolderwatchPath` as `watcher-go-<ts>.crawljob`. Returns `("", nil)` when there's nothing to merge (missing/empty file or no entries).

- [ ] **Step 1: Write the failing tests**

Append to `internal/jdownloader/writer_test.go`:

```go
func TestMergeMovesFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "out.crawljob")
	watch := filepath.Join(dir, "folderwatch")
	w := NewWriter(Config{File: file, FolderwatchPath: watch})

	if err := w.Add("pkg", dir, "https://src", []string{"https://mega.nz/a"}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	movedTo, err := w.Merge(1700000000)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if movedTo != filepath.Join(watch, "watcher-go-1700000000.crawljob") {
		t.Errorf("movedTo = %q", movedTo)
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Errorf("local file should be gone after merge")
	}
	if _, err := os.Stat(movedTo); err != nil {
		t.Errorf("destination file should exist: %v", err)
	}
}

func TestMergeEmptyIsNoop(t *testing.T) {
	dir := t.TempDir()
	w := NewWriter(Config{File: filepath.Join(dir, "missing.crawljob"), FolderwatchPath: filepath.Join(dir, "fw")})

	movedTo, err := w.Merge(1)
	if err != nil {
		t.Fatalf("Merge on missing file should not error: %v", err)
	}
	if movedTo != "" {
		t.Errorf("movedTo = %q, want empty", movedTo)
	}
}

func TestMergeRequiresFolderwatchPath(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "out.crawljob")
	w := NewWriter(Config{File: file}) // no FolderwatchPath

	_ = w.Add("pkg", dir, "https://src", []string{"https://mega.nz/a"})

	if _, err := w.Merge(1); err == nil {
		t.Errorf("Merge with no folderwatch_path should error")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/jdownloader/... -run 'TestMerge' -v`
Expected: FAIL — `Merge` undefined.

- [ ] **Step 3: Implement `Merge` and `moveFile`**

Append to `internal/jdownloader/writer.go` (add `io` to imports):

```go
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
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/jdownloader/... -v`
Expected: PASS (entire package).

- [ ] **Step 5: Lint and build**

Run: `golangci-lint run ./internal/jdownloader/...` then `go build -v .`
Expected: clean. Commit-ready.

---

### Task 5: `watcher crawljob merge` command

**Files:**
- Create: `cmd/watcher/crawljob.go`
- Modify: `cmd/watcher/main.go` (register the command)

**Interfaces:**
- Consumes: `jdownloader.LoadConfig`, `jdownloader.NewWriter`, `(*Writer).Merge` (Tasks 1–4).
- Produces: `func (cli *CliApplication) addCrawljobCommand()`.

- [ ] **Step 1: Implement the command**

Create `cmd/watcher/crawljob.go`:

```go
package watcher

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/DaRealFreak/watcher-go/internal/jdownloader"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/spf13/cobra"
)

// addCrawljobCommand adds the `crawljob` command group for the JDownloader handoff.
func (cli *CliApplication) addCrawljobCommand() {
	crawljobCmd := &cobra.Command{
		Use:   "crawljob",
		Short: "manage the JDownloader .crawljob handoff file",
	}

	var fileOverride, folderwatchOverride string
	mergeCmd := &cobra.Command{
		Use:   "merge",
		Short: "move the accumulated .crawljob file into JDownloader's folderwatch directory",
		Long: "moves the local .crawljob file (built up from external links watcher-go can't\n" +
			"download itself) into JDownloader's Folder Watch directory so JDownloader picks\n" +
			"it up and downloads each file into its post folder. The local file is consumed.",
		Run: func(_ *cobra.Command, _ []string) {
			cfg := jdownloader.LoadConfig()
			if fileOverride != "" {
				cfg.File = fileOverride
			}
			if folderwatchOverride != "" {
				cfg.FolderwatchPath = folderwatchOverride
			}

			movedTo, err := jdownloader.NewWriter(cfg).Merge(time.Now().Unix())
			raven.CheckError(err)

			if movedTo == "" {
				slog.Info("no crawljob entries to merge")
				return
			}
			slog.Info(fmt.Sprintf("merged crawljob into %s", movedTo))
		},
	}
	mergeCmd.Flags().StringVar(&fileOverride, "file", "", "override the local crawljob file path")
	mergeCmd.Flags().StringVar(&folderwatchOverride, "folderwatch", "", "override JDownloader's folderwatch directory")

	crawljobCmd.AddCommand(mergeCmd)
	cli.rootCmd.AddCommand(crawljobCmd)
}
```

- [ ] **Step 2: Register the command in `main.go`**

In `cmd/watcher/main.go`, inside `NewWatcherApplication()`, add the registration call alongside the other `app.add*Command()` calls (after `app.addProxyLimitsCommand()`):

```go
	app.addProxyLimitsCommand()
	app.addCrawljobCommand()
	app.addGenerateAutoCompletionCommand()
```

- [ ] **Step 3: Build and verify the command is wired**

Run: `go build -v .`
Then: `go run . crawljob merge --help`
Expected: build succeeds; help text for `merge` prints with `--file` and `--folderwatch` flags. (With no config, a real `crawljob merge` prints "no crawljob entries to merge" since the default file won't exist — safe no-op.)

- [ ] **Step 4: Lint**

Run: `golangci-lint run ./cmd/...`
Expected: clean. Commit-ready.

---

### Task 6: pawchive integration

**Files:**
- Modify: `internal/modules/pawchive/download.go` (gate at `getExternalLinks` ~line 149; loop ~lines 283–324)
- Test: `internal/modules/pawchive/download_test.go`

**Interfaces:**
- Consumes: `jdownloader.Enabled`, `jdownloader.Default`, `(*Writer).Blacklisted`, `(*Writer).Add`.

- [ ] **Step 1: Write the failing regression test for the gate change**

Append to `internal/modules/pawchive/download_test.go` (add `"github.com/spf13/viper"` to the import block):

```go
func TestGetExternalLinks_enabledByCrawljob(t *testing.T) {
	viper.Set("crawljob.enabled", true)
	defer viper.Set("crawljob.enabled", false)

	m := &pawchive{} // print/download both false; only crawljob enables collection
	post := &api.Post{
		User:    "u1",
		Content: "https://mega.nz/folder/abc",
	}

	links := m.getExternalLinks(post, nil)
	if len(links) != 1 || links[0] != "https://mega.nz/folder/abc" {
		t.Errorf("expected crawljob-enabled collection to return the link, got %v", links)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/modules/pawchive/... -run 'TestGetExternalLinks_enabledByCrawljob' -v`
Expected: FAIL — `getExternalLinks` early-returns (no links) because the gate doesn't yet consider crawljob.

- [ ] **Step 3: Update the `getExternalLinks` gate**

In `internal/modules/pawchive/download.go`, add the jdownloader import:

```go
	"github.com/DaRealFreak/watcher-go/internal/jdownloader"
```

Change the gate (currently):

```go
	if !m.settings.ExternalURLs.DownloadExternalItems && !m.settings.ExternalURLs.PrintExternalItems {
		return links
	}
```

to:

```go
	if !m.settings.ExternalURLs.DownloadExternalItems &&
		!m.settings.ExternalURLs.PrintExternalItems &&
		!jdownloader.Enabled() {
		return links
	}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/modules/pawchive/... -run 'TestGetExternalLinks' -v`
Expected: PASS (both the new test and the existing `TestGetExternalLinks*`).

- [ ] **Step 5: Route unparseable links to the crawljob**

In `internal/modules/pawchive/download.go`, replace the external-link loop body. Currently:

```go
	for _, externalURL := range externalLinks {
		if m.settings.ExternalURLs.PrintExternalItems {
			slog.Info(fmt.Sprintf("found external URL: \"%s\" in post \"%s\"", externalURL, webUrl), "module", m.Key)
		}

		if m.settings.ExternalURLs.DownloadExternalItems {
			if factory.CanParse(externalURL) {
				// ... existing native-download block ...
			} else {
				slog.Warn(fmt.Sprintf("unable to parse URL \"%s\" found in post \"%s\"", externalURL, webUrl), "module", m.Key)
			}
		}
	}
```

Insert a crawljob branch between the print and the `DownloadExternalItems` block, so unparseable links are routed (and `continue` so they don't also warn). Keep the existing native-download block and the `else` warn unchanged:

```go
	for _, externalURL := range externalLinks {
		if m.settings.ExternalURLs.PrintExternalItems {
			slog.Info(fmt.Sprintf("found external URL: \"%s\" in post \"%s\"", externalURL, webUrl), "module", m.Key)
		}

		// hand links we can't parse ourselves to JDownloader (independent of DownloadExternalItems)
		if !factory.CanParse(externalURL) {
			downloadFolder := path.Join(
				m.GetDownloadDirectory(),
				m.Key,
				fp.TruncateMaxLength(m.getSubFolder(item)),
				fp.TruncateMaxLength(postFolderPath),
			)
			pkg := fmt.Sprintf("%s - %s", m.Key, postFolderPath)
			if jdownloader.Default().Queue(m.Key, pkg, downloadFolder, webUrl, externalURL) {
				continue
			}
		}

		if m.settings.ExternalURLs.DownloadExternalItems {
			if factory.CanParse(externalURL) {
				// ... existing native-download block, unchanged ...
			} else {
				slog.Warn(fmt.Sprintf("unable to parse URL \"%s\" found in post \"%s\"", externalURL, webUrl), "module", m.Key)
			}
		}
	}
```

> NOTE: `Queue` returns false when crawljob is disabled or the link is blacklisted, so those links fall through to the existing `DownloadExternalItems` warn — preserving today's behavior when crawljob is off.

`postFolderPath`, `webUrl`, `item`, and `factory` are all already in scope at this loop (see `download.go`). `fp` and `path` are already imported in this file. `cj.Enabled()` is the `(*Writer).Enabled()` method added in Task 2.

- [ ] **Step 6: Run the whole module test suite**

Run: `go test ./internal/modules/pawchive/... -v`
Expected: PASS.

- [ ] **Step 7: Lint and build**

Run: `golangci-lint run ./internal/... && go build -v .`
Expected: clean. Commit-ready.

---

### Task 7: kemono integration

**Files:**
- Modify: `internal/modules/kemono/download.go` (gate at `getExternalLinks` ~line 345; loop ~lines 277–325)
- Test: `internal/modules/kemono/download_test.go` (create if absent)

**Interfaces:**
- Consumes: `jdownloader.Enabled`, `jdownloader.Default`, `(*Writer).Enabled`, `(*Writer).Blacklisted`, `(*Writer).Add`.

- [ ] **Step 1: Write the failing regression test for the gate change**

If `internal/modules/kemono/download_test.go` does not exist, create it; otherwise append. Use the kemono post type `api.PostRoot` (confirm the exact shape with `internal/modules/kemono/api`):

```go
package kemono

import (
	"testing"

	"github.com/DaRealFreak/watcher-go/internal/modules/kemono/api"
	"github.com/spf13/viper"
)

func TestGetExternalLinks_enabledByCrawljob(t *testing.T) {
	viper.Set("crawljob.enabled", true)
	defer viper.Set("crawljob.enabled", false)

	m := &kemono{} // print/download both false
	post := &api.PostRoot{}
	post.Post.Content = "grab https://mega.nz/folder/abc"

	links := m.getExternalLinks(post, nil)
	if len(links) != 1 || links[0] != "https://mega.nz/folder/abc" {
		t.Errorf("expected crawljob-enabled collection, got %v", links)
	}
}
```

> Before writing this test, open `internal/modules/kemono/api` and verify how `Content` and `User` are accessed on `PostRoot` (the explorer reported `post.Post.Content` and `post.Post.User`). Adjust field access to match the real struct; do not guess.

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/modules/kemono/... -run 'TestGetExternalLinks_enabledByCrawljob' -v`
Expected: FAIL — gate returns no links.

- [ ] **Step 3: Update the gate**

In `internal/modules/kemono/download.go`, add the import `"github.com/DaRealFreak/watcher-go/internal/jdownloader"` and change the `getExternalLinks` gate:

```go
	if !m.settings.ExternalURLs.DownloadExternalItems && !m.settings.ExternalURLs.PrintExternalItems {
		return links
	}
```

to:

```go
	if !m.settings.ExternalURLs.DownloadExternalItems &&
		!m.settings.ExternalURLs.PrintExternalItems &&
		!jdownloader.Enabled() {
		return links
	}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/modules/kemono/... -run 'TestGetExternalLinks' -v`
Expected: PASS.

- [ ] **Step 5: Route unparseable links to the crawljob**

In `internal/modules/kemono/download.go`, insert the same crawljob branch into the external-link loop, between the `PrintExternalItems` log and the `DownloadExternalItems` block. The kemono per-post folder uses `postFolderPath` (built at ~line 160 as `data.ID [ - title]`) and `m.getSubFolder(item)`:

```go
		// hand links we can't parse ourselves to JDownloader (independent of DownloadExternalItems)
		if !factory.CanParse(externalURL) {
			downloadFolder := path.Join(
				m.GetDownloadDirectory(),
				m.Key,
				fp.TruncateMaxLength(m.getSubFolder(item)),
				fp.TruncateMaxLength(postFolderPath),
			)
			pkg := fmt.Sprintf("%s - %s", m.Key, postFolderPath)
			if jdownloader.Default().Queue(m.Key, pkg, downloadFolder, webUrl, externalURL) {
				continue
			}
		}
```

This goes immediately after the `PrintExternalItems` log block and before `if m.settings.ExternalURLs.DownloadExternalItems {`. `factory`, `webUrl`, `item`, `postFolderPath` are in scope; `fp` and `path` are already imported. When `Queue` returns false (crawljob off or blacklisted), the link falls through to the existing `else` warn in the `DownloadExternalItems` block — same as today.

- [ ] **Step 6: Run the module test suite**

Run: `go test ./internal/modules/kemono/... -v`
Expected: PASS.

- [ ] **Step 7: Lint and build**

Run: `golangci-lint run ./internal/... && go build -v .`
Expected: clean. Commit-ready.

---

### Task 8: patreon integration

**Files:**
- Modify: `internal/modules/patreon/download.go` (external-link loop ~lines 87–125)

**Interfaces:**
- Consumes: `jdownloader.Default`, `(*Writer).Enabled`, `(*Writer).Blacklisted`, `(*Writer).Add`, `modules.GetModuleFactory().CanParse`.

**Notes:** Patreon collects `data.ExternalURLs` unconditionally during parse, so there is no `getExternalLinks` gate to change. Patreon's existing `DownloadExternalItems` branch calls `GetModuleFromURI` **without** a `CanParse` guard — this fatals (via `raven.CheckError`) on an unparseable URL. This task adds the guard, which both fixes that latent crash and provides the crawljob routing point. Patreon files have **no per-post subfolder**; they land in the creator folder, so the crawljob `downloadFolder` is the creator folder.

- [ ] **Step 1: Add the factory + crawljob routing and a `CanParse` guard**

In `internal/modules/patreon/download.go`, add the imports:

```go
	"github.com/DaRealFreak/watcher-go/internal/jdownloader"
	"github.com/DaRealFreak/watcher-go/internal/modules"
```

(`modules` may already be imported — check before adding to avoid a duplicate import.)

The loop currently is:

```go
	for _, externalURL := range data.ExternalURLs {
		if m.settings.ExternalURLs.PrintExternalItems {
			slog.Info(fmt.Sprintf("found external URL: \"%s\" in post \"%s\"", externalURL, data.PatreonURL), "module", m.Key)
		}

		if m.settings.ExternalURLs.DownloadExternalItems {
			module := modules.GetModuleFactory().GetModuleFromURI(externalURL)
			// ... existing native-download block ...
		}
	}
```

Restructure to add a factory handle, the crawljob branch, and a `CanParse` guard around the native block:

```go
	factory := modules.GetModuleFactory()
	for _, externalURL := range data.ExternalURLs {
		if m.settings.ExternalURLs.PrintExternalItems {
			slog.Info(fmt.Sprintf("found external URL: \"%s\" in post \"%s\"", externalURL, data.PatreonURL), "module", m.Key)
		}

		// hand links we can't parse ourselves to JDownloader (independent of DownloadExternalItems)
		if !factory.CanParse(externalURL) {
			downloadFolder := path.Join(
				m.GetDownloadDirectory(),
				m.Key,
				strings.TrimSpace(fmt.Sprintf("%d_%s", data.CreatorID, data.CreatorName)),
			)
			pkg := fmt.Sprintf("%s - %d", m.Key, data.PostID)
			if !jdownloader.Default().Queue(m.Key, pkg, downloadFolder, data.PatreonURL, externalURL) {
				// crawljob off or blacklisted: warn but DON'T fall through to the native
				// block below, which would fatally call GetModuleFromURI on an unparseable URL.
				if m.settings.ExternalURLs.DownloadExternalItems {
					slog.Warn(fmt.Sprintf("unable to parse URL \"%s\" found in post \"%s\"", externalURL, data.PatreonURL), "module", m.Key)
				}
			}
			continue
		}

		if m.settings.ExternalURLs.DownloadExternalItems {
			module := modules.GetModuleFactory().GetModuleFromURI(externalURL)
			// ... existing native-download block, unchanged ...
		}
	}
```

Verify `path` and `strings` are imported in this file (they are used by the existing file-download paths above). `data.CreatorID` (int), `data.CreatorName` (string), `data.PostID` (int), `data.PatreonURL` (string) are fields on the loop's `data` struct.

- [ ] **Step 2: Build and run patreon tests**

Run: `go build -v . && go test ./internal/modules/patreon/... -v`
Expected: build succeeds; existing tests pass (no patreon unit test targets this inline loop — it requires full API/DbIO context — so coverage here is build + manual verification, plus the shared-package tests from Tasks 1–4).

- [ ] **Step 3: Lint**

Run: `golangci-lint run ./internal/...`
Expected: clean. Commit-ready.

---

### Task 9: pixiv integration

**Files:**
- Modify: `internal/modules/pixiv/download.go` (caption external-link block ~lines 58–81)

**Interfaces:**
- Consumes: `jdownloader.Default`, `(*Writer).Enabled`, `(*Writer).Blacklisted`, `(*Writer).Add`.

**Notes:** pixiv is the outlier — it only **prints** caption links today (no native external download, settings field is `ExternalUrls.PrintExternalItems`). All non-fanbox/discord caption links are "unparseable" from watcher-go's perspective, so they all route to the crawljob when enabled. Illustration files are flat under `path.Join(GetDownloadDirectory(), m.Key, data.DownloadTag)`, so that is the crawljob `downloadFolder`.

- [ ] **Step 1: Gate the caption block on crawljob too, and route links**

In `internal/modules/pixiv/download.go`, add the import:

```go
	"github.com/DaRealFreak/watcher-go/internal/jdownloader"
```

The block currently is:

```go
			if m.settings.ExternalUrls.PrintExternalItems {
				checkedIllustration := item
				if item.Caption == "" {
					detail, detailErr := m.mobileAPI.GetIllustDetail(item.ID)
					if detailErr != nil {
						slog.Error(fmt.Sprintf("failed to get illustration details for ID %d: %v", item.ID, detailErr), "module", m.Key)
					}
					checkedIllustration = detail.Illustration
				}
				links := linkfinder.GetLinks(checkedIllustration.Caption)
				for _, link := range links {
					if strings.Contains(link, ".fanbox.cc/") || strings.Contains(link, "discord.gg/") {
						continue
					}
					link = strings.Replace(link, "http://", "https://", 1)
					slog.Info(fmt.Sprintf("found external URL: \"%s\" in post \"https://www.pixiv.net/en/artworks/%d\"", link, checkedIllustration.ID), "module", m.Key)
				}
			}
```

Change the outer `if` to also enter when crawljob is enabled, keep the print gated on `PrintExternalItems`, and add the crawljob route per link:

```go
			if m.settings.ExternalUrls.PrintExternalItems || jdownloader.Enabled() {
				checkedIllustration := item
				if item.Caption == "" {
					detail, detailErr := m.mobileAPI.GetIllustDetail(item.ID)
					if detailErr != nil {
						slog.Error(fmt.Sprintf("failed to get illustration details for ID %d: %v", item.ID, detailErr), "module", m.Key)
					}
					checkedIllustration = detail.Illustration
				}
				links := linkfinder.GetLinks(checkedIllustration.Caption)
				for _, link := range links {
					if strings.Contains(link, ".fanbox.cc/") || strings.Contains(link, "discord.gg/") {
						continue
					}
					link = strings.Replace(link, "http://", "https://", 1)

					if m.settings.ExternalUrls.PrintExternalItems {
						slog.Info(fmt.Sprintf("found external URL: \"%s\" in post \"https://www.pixiv.net/en/artworks/%d\"", link, checkedIllustration.ID), "module", m.Key)
					}

					// pixiv never downloads caption links natively, so every link is a crawljob
					// candidate. Queue no-ops when crawljob is disabled or the host is blacklisted.
					downloadFolder := path.Join(m.GetDownloadDirectory(), m.Key, data.DownloadTag)
					pkg := fmt.Sprintf("%s - %d", m.Key, checkedIllustration.ID)
					webUrl := fmt.Sprintf("https://www.pixiv.net/en/artworks/%d", checkedIllustration.ID)
					jdownloader.Default().Queue(m.Key, pkg, downloadFolder, webUrl, link)
				}
			}
```

Verify `path` is imported in `internal/modules/pixiv/download.go` (it is — used by the download paths). `data.DownloadTag` is the per-illustration folder segment; `data` is in scope in this loop.

- [ ] **Step 2: Build and run pixiv tests**

Run: `go build -v . && go test ./internal/modules/pixiv/... -v`
Expected: build succeeds; existing tests pass. (As with patreon, this inline block isn't unit-testable without the pixiv API; coverage is build + manual verification plus the shared-package tests.)

- [ ] **Step 3: Lint**

Run: `golangci-lint run ./internal/...`
Expected: clean. Commit-ready.

---

### Task 10: Full verification & docs

**Files:**
- Modify: `CLAUDE.md` (optional — note the new command) — only if the user wants it; otherwise skip.

- [ ] **Step 1: Full test suite**

Run: `go test ./...`
Expected: PASS across the repo.

- [ ] **Step 2: Full build & lint**

Run: `go build -v . && golangci-lint run`
Expected: clean.

- [ ] **Step 3: Manual smoke test (real config)**

Add to your config YAML:

```yaml
crawljob:
  enabled: true
  file: ./watcher-go.crawljob
  folderwatch_path: C:\Users\<you>\AppData\Local\JDownloader 2.0\folderwatch
  blacklist: [discord.gg, patreon.com, t.me]
  auto_start: true
  auto_confirm: true
```

Run a tracked pawchive item that has mega/mediafire links, confirm `./watcher-go.crawljob` is created with one entry per post and correct absolute `downloadFolder`s. Then run `watcher crawljob merge` and confirm the file moves into `folderwatch` as `watcher-go-<ts>.crawljob` and JDownloader ingests it.

- [ ] **Step 4: Hand off to user for commit**

Per the user's git workflow, do not commit. Summarize the changes and let the user stage/commit.

---

## Self-Review Notes

- **Spec coverage:** shared package (Tasks 1–4) ✔; command (Task 5) ✔; per-module integration with "only unparseable links" routing (Tasks 6–9) ✔; global config block + blacklist + auto flags (Task 1) ✔; absolute-path resolution (Task 3) ✔; move-on-merge with unique name (Task 4) ✔; concurrency mutex (Tasks 2–4) ✔; tests for writer/blacklist/merge + module gate (Tasks 1–4, 6, 7) ✔.
- **Known coverage gap (surfaced, not hidden):** patreon and pixiv inline routing (Tasks 8–9) have no unit test because the loops require full API/DbIO context the test suite doesn't stub. They are covered by `go build`, the shared-package tests, and the Task 10 manual smoke test. The gate change *is* unit-tested for pawchive and kemono, which share the routing block's structure.
- **`Writer.Enabled()`** is defined in Task 2 (where the Writer is built) and first consumed by the module integrations in Tasks 6–9.
- **Type consistency:** `Add(packageName, downloadFolder, sourceURL string, links []string) error`, `Queue(moduleKey, packageName, downloadFolder, sourceURL, link string) bool`, `Merge(ts int64) (string, error)`, `Blacklisted(string) bool`, `Enabled()` (package func) and `(*Writer).Enabled()` (method) are used consistently across tasks. The four module integrations call `jdownloader.Default().Queue(...)` (Tasks 6–9), which centralizes the enabled/blacklist/Add/log logic so each module only computes its own `downloadFolder`/`packageName`.
