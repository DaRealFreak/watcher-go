//go:build !windows
// +build !windows

package fp

import (
	"strings"
)

// TruncateMaxLength ensures the string does not exceed the maximum path length (4096)
// minus any reserved characters. The reserved parameter is optional and defaults to 0.
func TruncateMaxLength(s string, reserved ...int) string {
	reserve := 0
	if len(reserved) > 0 {
		reserve = reserved[0]
	}
	allowed := 4096 - reserve

	if len(s) < allowed {
		return s
	}

	if idx := strings.LastIndex(s[:allowed], " "); idx != -1 {
		return s[:idx]
	}

	return s[:allowed]
}
