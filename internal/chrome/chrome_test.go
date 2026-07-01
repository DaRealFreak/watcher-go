package chrome

import (
	"archive/zip"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCftPlatformFor(t *testing.T) {
	cases := []struct {
		goos, goarch string
		want         string
		wantErr      bool
	}{
		{"windows", "amd64", "win64", false},
		{"windows", "386", "win32", false},
		{"darwin", "amd64", "mac-x64", false},
		{"darwin", "arm64", "mac-arm64", false},
		{"linux", "amd64", "linux64", false},
		{"linux", "arm64", "", true},
		{"plan9", "amd64", "", true},
	}
	for _, c := range cases {
		got, err := cftPlatformFor(c.goos, c.goarch)
		if c.wantErr {
			if err == nil {
				t.Errorf("cftPlatformFor(%s,%s): expected error, got %q", c.goos, c.goarch, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("cftPlatformFor(%s,%s): unexpected error %v", c.goos, c.goarch, err)
			continue
		}
		if got != c.want {
			t.Errorf("cftPlatformFor(%s,%s) = %q, want %q", c.goos, c.goarch, got, c.want)
		}
	}
}

func TestChromeExeRelPath(t *testing.T) {
	cases := map[string]string{
		"win64":     "chrome.exe",
		"win32":     "chrome.exe",
		"linux64":   "chrome",
		"mac-x64":   filepath.Join("Google Chrome for Testing.app", "Contents", "MacOS", "Google Chrome for Testing"),
		"mac-arm64": filepath.Join("Google Chrome for Testing.app", "Contents", "MacOS", "Google Chrome for Testing"),
	}
	for platform, want := range cases {
		if got := chromeExeRelPath(platform); got != want {
			t.Errorf("chromeExeRelPath(%q) = %q, want %q", platform, got, want)
		}
	}
}

func TestSystemChromeCandidatesNonEmpty(t *testing.T) {
	for _, goos := range []string{"windows", "darwin", "linux"} {
		if len(systemChromeCandidates(goos)) == 0 {
			t.Errorf("systemChromeCandidates(%q) returned no candidates", goos)
		}
	}
}

func TestFindConfigPath(t *testing.T) {
	// existing file is returned as-is
	f := filepath.Join(t.TempDir(), "mychrome")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Find(f)
	if err != nil || got != f {
		t.Errorf("Find(existing) = %q, %v; want %q, nil", got, err, f)
	}

	// missing configured path is a hard error (no silent fallback)
	if _, err := Find(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Error("Find(missing config path): expected error, got nil")
	}
}

func TestExtractZipPreservesModeAndSymlinks(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "test.zip")

	buildTestZip(t, zipPath)

	dest := filepath.Join(dir, "out")
	if err := extractZip(zipPath, dest); err != nil {
		t.Fatalf("extractZip: %v", err)
	}

	// regular executable: exec bit preserved (unix only; Windows ignores mode bits)
	exe := filepath.Join(dest, "chrome-linux64", "chrome")
	info, err := os.Stat(exe)
	if err != nil {
		t.Fatalf("stat extracted exe: %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0 {
		t.Errorf("expected executable bit on %s, got mode %v", exe, info.Mode())
	}

	// nested file extracted
	if !fileExists(filepath.Join(dest, "chrome-linux64", "resources", "data.pak")) {
		t.Error("expected nested resource file to be extracted")
	}

	// symlink preserved as a symlink (unix only)
	if runtime.GOOS != "windows" {
		link := filepath.Join(dest, "chrome-linux64", "current")
		li, err := os.Lstat(link)
		if err != nil {
			t.Fatalf("lstat symlink: %v", err)
		}
		if li.Mode()&os.ModeSymlink == 0 {
			t.Errorf("expected %s to be a symlink, got mode %v", link, li.Mode())
		}
	}
}

func TestExtractZipRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "evil.zip")

	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("../escape.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte("pwned"))
	_ = zw.Close()
	_ = f.Close()

	if err := extractZip(zipPath, filepath.Join(dir, "out")); err == nil {
		t.Error("expected extractZip to reject path traversal entry")
	}
}

// buildTestZip writes a small archive mimicking a chrome-linux64 layout with an
// executable, a nested file, and a symlink (stored with unix mode bits).
func buildTestZip(t *testing.T, zipPath string) {
	t.Helper()
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	zw := zip.NewWriter(f)
	defer func() { _ = zw.Close() }()

	addFile := func(name string, mode os.FileMode, content string) {
		hdr := &zip.FileHeader{Name: name, Method: zip.Deflate}
		hdr.SetMode(mode)
		w, err := zw.CreateHeader(hdr)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}

	addFile("chrome-linux64/chrome", 0o755, "#!/bin/sh\n")
	addFile("chrome-linux64/resources/data.pak", 0o644, "data")
	// Windows can't create symlinks without elevation, and the real Chrome-for-Testing
	// win64 archive contains none (only the macOS .app bundle does), so only exercise
	// symlink extraction on unix.
	if runtime.GOOS != "windows" {
		addFile("chrome-linux64/current", os.ModeSymlink|0o777, "chrome")
	}
}
