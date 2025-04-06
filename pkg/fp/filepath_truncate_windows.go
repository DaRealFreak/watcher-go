//go:build windows

package fp

import (
	"strings"
	"syscall"
)

// TruncateMaxLength checks for length of the passed path part to ensure the max path length
func TruncateMaxLength(s string, reserved ...int) string {
	// Use a default reserved value of 5 if none is provided.
	reserve := 0
	if len(reserved) > 0 {
		reserve = reserved[0]
	}

	// -5 for prefixes and null byte ending
	reserve += 5

	// Calculate the maximum allowed length for the string.
	maxAllowed := syscall.MAX_PATH - reserve

	// If the string is already within the allowed length, return it unchanged.
	if len(s) < maxAllowed {
		return s
	}

	// Prefer splitting on the last space within the allowed length.
	if idx := strings.LastIndex(s[:maxAllowed], " "); idx != -1 {
		return s[:idx]
	}

	// Otherwise, do a hard cut.
	return s[:maxAllowed]
}
