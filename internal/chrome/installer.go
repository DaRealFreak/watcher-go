package chrome

import (
	"archive/zip"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// latestStableURL returns the current Chrome-for-Testing stable version string.
	latestStableURL = "https://googlechromelabs.github.io/chrome-for-testing/LATEST_RELEASE_STABLE"
	// downloadURLTemplate is formatted with (version, platform, platform).
	downloadURLTemplate = "https://storage.googleapis.com/chrome-for-testing-public/%s/%s/chrome-%s.zip"
)

// Installer downloads and caches a Chrome-for-Testing build.
type Installer struct {
	// Version is the Chrome-for-Testing version to install, or "latest".
	Version string
	// CacheDir is the base directory downloads are extracted into.
	CacheDir string
	// Client is the HTTP client used for the download (defaults to a 10 min client).
	Client *http.Client
}

// NewInstaller returns an Installer that fetches the latest stable build into
// the user cache directory (~/.cache/watcher-go/chrome on unix,
// %LocalAppData%/watcher-go/chrome on Windows).
func NewInstaller() *Installer {
	return &Installer{
		Version:  "latest",
		CacheDir: defaultCacheDir(),
		Client:   &http.Client{Timeout: 10 * time.Minute},
	}
}

func defaultCacheDir() string {
	base, err := os.UserCacheDir()
	if err != nil || base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "watcher-go", "chrome")
}

// Install ensures a Chrome-for-Testing build is present and returns the path to
// its executable, downloading and extracting it if necessary.
func (i *Installer) Install() (string, error) {
	platform, err := cftPlatform()
	if err != nil {
		return "", err
	}

	version := i.Version
	if version == "" || strings.EqualFold(version, "latest") {
		version, err = i.latestStableVersion()
		if err != nil {
			return "", fmt.Errorf("could not resolve latest Chrome for Testing version: %w", err)
		}
		slog.Debug(fmt.Sprintf("resolved latest Chrome for Testing version: %s", version))
	}

	extractDir := filepath.Join(i.CacheDir, version)
	exePath := filepath.Join(extractDir, fmt.Sprintf("chrome-%s", platform), chromeExeRelPath(platform))
	if fileExists(exePath) {
		slog.Debug(fmt.Sprintf("using cached Chrome for Testing: %s", exePath))
		return exePath, nil
	}

	url := fmt.Sprintf(downloadURLTemplate, version, platform, platform)
	slog.Info(fmt.Sprintf("downloading Chrome for Testing %s (%s) ...", version, platform))
	if err = i.downloadAndExtract(url, extractDir); err != nil {
		return "", err
	}

	if !fileExists(exePath) {
		return "", fmt.Errorf("chrome executable not found after extraction: %s", exePath)
	}
	slog.Info(fmt.Sprintf("installed Chrome for Testing to %s", exePath))
	return exePath, nil
}

func (i *Installer) latestStableVersion() (string, error) {
	resp, err := i.httpClient().Get(latestStableURL)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d fetching latest version", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		return "", err
	}
	version := strings.TrimSpace(string(body))
	if version == "" {
		return "", fmt.Errorf("empty version response")
	}
	return version, nil
}

// downloadAndExtract streams the zip to a temp file and extracts it into destDir.
func (i *Installer) downloadAndExtract(url, destDir string) error {
	resp, err := i.httpClient().Get(url)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d downloading %s", resp.StatusCode, url)
	}

	tmp, err := os.CreateTemp("", "chrome-cft-*.zip")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	_, err = io.Copy(tmp, resp.Body)
	closeErr := tmp.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}

	return extractZip(tmpName, destDir)
}

func (i *Installer) httpClient() *http.Client {
	if i.Client != nil {
		return i.Client
	}
	return http.DefaultClient
}

// extractZip extracts a zip archive into destDir, preserving the unix
// permission bits and symlinks stored in each entry. The standard
// zip.Reader/extractall equivalents drop the mode held in external_attr, which
// leaves the extracted chrome binary non-executable on Linux/macOS.
func extractZip(zipPath, destDir string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()

	if err = os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	for _, file := range reader.File {
		if err = extractZipEntry(file, destDir); err != nil {
			return err
		}
	}
	return nil
}

func extractZipEntry(file *zip.File, destDir string) error {
	target, err := sanitizeZipPath(destDir, file.Name)
	if err != nil {
		return err
	}

	mode := file.Mode()

	switch {
	case mode&os.ModeSymlink != 0:
		return writeZipSymlink(file, target)
	case file.FileInfo().IsDir():
		return os.MkdirAll(target, 0o755)
	default:
		return writeZipFile(file, target, mode)
	}
}

// sanitizeZipPath joins name onto destDir while rejecting entries that would
// escape the destination directory (zip-slip protection).
func sanitizeZipPath(destDir, name string) (string, error) {
	target := filepath.Join(destDir, name) //nolint:gosec // guarded by the prefix check below
	cleanDest := filepath.Clean(destDir) + string(os.PathSeparator)
	if !strings.HasPrefix(filepath.Clean(target)+string(os.PathSeparator), cleanDest) {
		return "", fmt.Errorf("zip entry escapes destination: %s", name)
	}
	return target, nil
}

func writeZipFile(file *zip.File, target string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}

	src, err := file.Open()
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()

	perm := mode.Perm()
	if perm == 0 {
		perm = 0o644
	}
	dst, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return err
	}

	if _, err = io.Copy(dst, src); err != nil { //nolint:gosec // trusted Google CDN archive
		_ = dst.Close()
		return err
	}
	if err = dst.Close(); err != nil {
		return err
	}
	// re-apply perm bits explicitly (umask may have masked them at create time)
	return os.Chmod(target, perm)
}

func writeZipSymlink(file *zip.File, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}

	src, err := file.Open()
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()

	linkTarget, err := io.ReadAll(src)
	if err != nil {
		return err
	}

	_ = os.Remove(target)
	return os.Symlink(string(linkTarget), target)
}
