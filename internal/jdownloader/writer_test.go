package jdownloader

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

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
