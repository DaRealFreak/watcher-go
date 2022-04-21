// +build windows

package models

import (
	"strings"
	"syscall"
)

// TruncateMaxLength checks for length of the passed path part to ensure the max path length
func (t Module) TruncateMaxLength(s string) string {
	if syscall.MAX_PATH > len(s) {
		return s
	}
	return s[:strings.LastIndex(s[:syscall.MAX_PATH], " ")]
}
