package fp

import (
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

// SanitizePath replaces reserved characters https://en.wikipedia.org/wiki/Filename#Reserved_characters_and_words
// and trims the result
func SanitizePath(path string, allowSeparator bool) string {
	var reservedCharacters *regexp.Regexp
	if allowSeparator {
		reservedCharacters = regexp.MustCompile("[:\"*?<>|]+")
	} else {
		reservedCharacters = regexp.MustCompile("[\\\\/:\"*?<>|]+")
	}

	// Replace escape sequences aside from \t, since we want to replace it with a space
	escapeSequences := []string{"\b", "\n", "\r", "\f", "\v"}
	for _, seq := range escapeSequences {
		path = strings.ReplaceAll(path, seq, "")
	}

	path = reservedCharacters.ReplaceAllString(path, "_")
	// replace tabulators with spaces
	path = strings.ReplaceAll(path, "\t", " ")

	// replace duplicate underscores, dots and spaces with one
	for strings.Contains(path, "__") || strings.Contains(path, "..") || strings.Contains(path, "  ") {
		path = strings.ReplaceAll(path, "__", "_")
		path = strings.ReplaceAll(path, "..", ".")
		path = strings.ReplaceAll(path, "  ", " ")
	}

	// trim leading and trailing underscores
	return strings.Trim(path, "_. ")
}

// GetFileName retrieves the file name of a passed uri
func GetFileName(uri string) string {
	parsedURI, parsedErr := url.Parse(uri)
	if parsedErr != nil {
		// fallback to filepath on f.e. invalid escape errors since they don't apply to filenames
		_, file := filepath.Split(uri)
		return strings.TrimSuffix(file, filepath.Ext(file))
	}
	return filepath.Base(parsedURI.Path)
}

// GetFileExtension retrieves the file extension of a passed uri
func GetFileExtension(uri string) string {
	parsedURI, parsedErr := url.Parse(uri)
	if parsedErr != nil {
		// fallback to filepath on f.e. invalid escape errors since they don't apply to filenames
		_, file := filepath.Split(uri)
		return filepath.Ext(file)
	}
	return filepath.Ext(parsedURI.Path)
}
