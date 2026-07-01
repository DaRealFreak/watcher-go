package chrome

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Instance is a launched Chrome process with an open DevTools endpoint.
type Instance struct {
	cmd     *exec.Cmd
	dataDir string
	// PageWSURL is the CDP WebSocket URL of the initial page target.
	PageWSURL string
}

// LaunchOptions configures a Chrome launch for CDP automation.
type LaunchOptions struct {
	// ExecPath is the Chrome/Chromium executable (required).
	ExecPath string
	// Headless runs Chrome with the new headless mode when true.
	Headless bool
	// UserAgent overrides the browser user agent when non-empty.
	UserAgent string
	// InitialURL is opened on startup (defaults to about:blank).
	InitialURL string
}

// Launch starts Chrome with a fresh, throwaway profile and a DevTools endpoint,
// returning an Instance whose PageWSURL can be driven over CDP.
//
// It deliberately avoids the --enable-automation switch (which would set
// navigator.webdriver); combined with a driver that never calls Runtime.enable,
// this keeps the session from tripping CDP-based bot detection.
func Launch(opts LaunchOptions) (*Instance, error) {
	if opts.ExecPath == "" {
		return nil, fmt.Errorf("chrome: no executable provided to Launch")
	}

	dataDir, err := os.MkdirTemp("", "watcher-chrome-*")
	if err != nil {
		return nil, err
	}

	initialURL := opts.InitialURL
	if initialURL == "" {
		initialURL = "about:blank"
	}

	args := []string{
		"--remote-debugging-port=0",
		"--user-data-dir=" + dataDir,
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-blink-features=AutomationControlled",
		"--window-size=1280,900",
	}
	if opts.Headless {
		args = append(args, "--headless=new", "--disable-gpu")
	}
	if opts.UserAgent != "" {
		args = append(args, "--user-agent="+opts.UserAgent)
	}
	args = append(args, initialURL)

	cmd := exec.Command(opts.ExecPath, args...)
	if err = cmd.Start(); err != nil {
		_ = os.RemoveAll(dataDir)
		return nil, err
	}

	inst := &Instance{cmd: cmd, dataDir: dataDir}

	port, err := waitDevToolsPort(filepath.Join(dataDir, "DevToolsActivePort"))
	if err != nil {
		_ = inst.Close()
		return nil, err
	}

	wsURL, err := findPageWS(port)
	if err != nil {
		_ = inst.Close()
		return nil, err
	}
	inst.PageWSURL = wsURL

	return inst, nil
}

// Close terminates the browser process and removes its throwaway profile.
func (i *Instance) Close() error {
	if i.cmd != nil && i.cmd.Process != nil {
		_ = i.cmd.Process.Kill()
		_, _ = i.cmd.Process.Wait()
	}
	if i.dataDir != "" {
		_ = os.RemoveAll(i.dataDir)
	}
	return nil
}

// waitDevToolsPort reads the port Chrome writes to DevToolsActivePort once its
// remote debugging endpoint is ready.
func waitDevToolsPort(portFile string) (string, error) {
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if b, err := os.ReadFile(portFile); err == nil {
			line := strings.SplitN(strings.TrimSpace(string(b)), "\n", 2)[0]
			if line != "" {
				return line, nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return "", fmt.Errorf("chrome DevTools port did not become available")
}

// findPageWS discovers the WebSocket URL of the first page target.
func findPageWS(port string) (string, error) {
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://127.0.0.1:" + port + "/json")
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()

			var targets []struct {
				Type string `json:"type"`
				WS   string `json:"webSocketDebuggerUrl"`
			}
			if json.Unmarshal(body, &targets) == nil {
				for _, t := range targets {
					if t.Type == "page" && t.WS != "" {
						return t.WS, nil
					}
				}
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return "", fmt.Errorf("no chrome page target found on debugging endpoint")
}
