package watcher

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/raven"
)

// parsedCookie holds the parsed fields from a Netscape cookie line
type parsedCookie struct {
	domain     string
	name       string
	value      string
	expiration string
}

// ImportCookiesByURI reads a Netscape format cookie file and imports all cookies
func (app *Watcher) ImportCookiesByURI(filePath string, uri string) {
	file, err := os.Open(filePath)
	raven.CheckError(err)

	defer raven.CheckClosure(file)

	cookies := parseNetscapeCookies(bufio.NewScanner(file))
	app.importParsedCookies(cookies, uri)
}

// ImportCookiesFromClipboardByURI reads Netscape format cookie data from the clipboard and imports it
func (app *Watcher) ImportCookiesFromClipboardByURI(uri string) {
	clipboardContent, err := readClipboard()
	raven.CheckError(err)

	cookies := parseNetscapeCookies(bufio.NewScanner(strings.NewReader(clipboardContent)))
	app.importParsedCookies(cookies, uri)
}

// importParsedCookies resolves the module and imports the parsed cookies
func (app *Watcher) importParsedCookies(cookies []parsedCookie, uri string) {
	if len(cookies) == 0 {
		slog.Warn("no cookies found to import")
		return
	}

	// resolve module from explicit URL or auto-detect from first cookie domain
	var module *models.Module
	if uri != "" {
		module = app.ModuleFactory.GetModuleFromURI(uri)
	} else {
		domain := strings.TrimPrefix(cookies[0].domain, ".")
		module = app.ModuleFactory.GetModuleFromURI("https://" + domain)
	}

	imported := 0
	for _, c := range cookies {
		if existing := app.DbCon.GetCookie(c.name, module); existing != nil {
			app.DbCon.UpdateCookie(c.name, c.value, c.expiration, module)
		} else {
			app.DbCon.GetFirstOrCreateCookie(c.name, c.value, c.expiration, module)
		}

		slog.Info(fmt.Sprintf("imported cookie \"%s\" for module %s", c.name, module.ModuleKey()))
		imported++
	}

	slog.Info(fmt.Sprintf("imported %d cookies for module %s", imported, module.ModuleKey()))
}

// parseNetscapeCookies parses Netscape format cookie lines from a scanner
func parseNetscapeCookies(scanner *bufio.Scanner) []parsedCookie {
	var cookies []parsedCookie

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Split(line, "\t")
		if len(fields) < 7 {
			slog.Warn(fmt.Sprintf("skipping malformed cookie line: %s", line))
			continue
		}

		// Netscape format: domain flag path secure expiration name value
		expiration := ""
		if expUnix, err := strconv.ParseInt(fields[4], 10, 64); err == nil && expUnix > 0 {
			expiration = time.Unix(expUnix, 0).Format(time.RFC3339)
		}

		cookies = append(cookies, parsedCookie{
			domain:     fields[0],
			name:       fields[5],
			value:      fields[6],
			expiration: expiration,
		})
	}

	raven.CheckError(scanner.Err())
	return cookies
}

// readClipboard reads text content from the system clipboard using platform-native commands
func readClipboard() (string, error) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("powershell", "-command", "Get-Clipboard")
	case "darwin":
		cmd = exec.Command("pbpaste")
	default:
		// Linux: try xclip first, fall back to xsel
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard", "-o")
		} else if _, err = exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--output")
		} else {
			return "", fmt.Errorf("no clipboard tool found (install xclip or xsel)")
		}
	}

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to read clipboard: %w", err)
	}

	return string(output), nil
}
