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

	path = reservedCharacters.ReplaceAllString(path, "_")
	path = strings.ReplaceAll(path, "\t", " ")
	for strings.Contains(path, "__") {
		path = strings.Replace(path, "__", "_", -1)
	}

	for strings.Contains(path, "..") {
		path = strings.Replace(path, "..", ".", -1)
	}

	path = strings.Trim(path, "_")

	return strings.Trim(path, " ")
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
