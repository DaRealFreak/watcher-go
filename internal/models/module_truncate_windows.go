//go:build windows
// +build windows

package models

import (
	"strings"
	"syscall"
)

// TruncateMaxLength checks for length of the passed path part to ensure the max path length
func (t Module) TruncateMaxLength(s string) string {
	// -5 for prefixes and null byte ending
	if syscall.MAX_PATH-5 > len(s) {
		return s
	}

	if strings.Contains(s, " ") {
		// prefer splitting the string on the last space
		return s[:strings.LastIndex(s[:syscall.MAX_PATH-5], " ")]
	} else {
		// else hard cut the string
		return s[:syscall.MAX_PATH-5]
	}
}
