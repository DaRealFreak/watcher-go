// Package chrome locates a usable Chrome/Chromium executable, downloading a
// Chrome-for-Testing build on demand when none is installed on the system.
//
// It is used by modules that need a real browser (e.g. to pass JavaScript-based
// bot protection during login) while keeping the rest of their traffic on the
// lightweight TLS HTTP client.
package chrome

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"log/slog"
)

// Find resolves a usable Chrome/Chromium executable path.
//
// Resolution order:
//  1. configPath, if non-empty and it exists on disk
//  2. a Chrome/Chromium/Edge install found in the usual system locations
//  3. a previously downloaded Chrome-for-Testing build in the cache
//  4. a freshly downloaded Chrome-for-Testing build (~150-200 MB, one time)
func Find(configPath string) (string, error) {
	if configPath != "" {
		if fileExists(configPath) {
			return configPath, nil
		}
		return "", fmt.Errorf("configured browser_path does not exist: %s", configPath)
	}

	if path := findSystemChrome(runtime.GOOS); path != "" {
		slog.Debug(fmt.Sprintf("using system browser: %s", path))
		return path, nil
	}

	slog.Info("no system Chrome/Chromium found, falling back to Chrome for Testing")
	return NewInstaller().Install()
}

// findSystemChrome returns the first existing browser executable from the
// platform's candidate locations, or "" if none is found.
func findSystemChrome(goos string) string {
	for _, candidate := range systemChromeCandidates(goos) {
		if filepath.IsAbs(candidate) {
			if fileExists(candidate) {
				return candidate
			}
			continue
		}
		// bare command name: resolve via PATH (Linux/macOS package installs)
		if resolved, err := exec.LookPath(candidate); err == nil {
			return resolved
		}
	}
	return ""
}

// systemChromeCandidates returns the ordered list of executables to probe for
// the given OS. Absolute paths are checked for existence; bare names are looked
// up on PATH.
func systemChromeCandidates(goos string) []string {
	switch goos {
	case "windows":
		var out []string
		for _, base := range []string{
			os.Getenv("ProgramFiles"),
			os.Getenv("ProgramFiles(x86)"),
			os.Getenv("LocalAppData"),
		} {
			if base == "" {
				continue
			}
			out = append(out,
				filepath.Join(base, "Google", "Chrome", "Application", "chrome.exe"),
				filepath.Join(base, "Chromium", "Application", "chrome.exe"),
				filepath.Join(base, "Microsoft", "Edge", "Application", "msedge.exe"),
			)
		}
		return out
	case "darwin":
		return []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
		}
	default: // linux and other unix
		return []string{
			"google-chrome",
			"google-chrome-stable",
			"chromium",
			"chromium-browser",
			"microsoft-edge",
			"microsoft-edge-stable",
		}
	}
}

// cftPlatform returns the Chrome-for-Testing platform token for the running host.
func cftPlatform() (string, error) {
	return cftPlatformFor(runtime.GOOS, runtime.GOARCH)
}

// cftPlatformFor maps a Go OS/arch pair to a Chrome-for-Testing platform token.
func cftPlatformFor(goos, goarch string) (string, error) {
	switch goos {
	case "windows":
		if goarch == "386" {
			return "win32", nil
		}
		return "win64", nil
	case "darwin":
		if goarch == "arm64" {
			return "mac-arm64", nil
		}
		return "mac-x64", nil
	case "linux":
		if goarch == "amd64" {
			return "linux64", nil
		}
		return "", fmt.Errorf("chrome-for-testing has no build for linux/%s", goarch)
	default:
		return "", fmt.Errorf("chrome-for-testing is unavailable for %s/%s", goos, goarch)
	}
}

// chromeExeRelPath returns the browser executable path relative to the extracted
// chrome-<platform> directory for the given platform token.
func chromeExeRelPath(platform string) string {
	switch {
	case strings.HasPrefix(platform, "win"):
		return "chrome.exe"
	case strings.HasPrefix(platform, "mac"):
		return filepath.Join(
			"Google Chrome for Testing.app", "Contents", "MacOS", "Google Chrome for Testing",
		)
	default:
		return "chrome"
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
