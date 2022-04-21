// +build !windows

package models

import (
	"strings"
)

// TruncateMaxLength checks for length of the passed path part to ensure the max path length
func (t Module) TruncateMaxLength(s string) string {
	if 4096 > len(s) {
		return s
	}
	return s[:strings.LastIndex(s[:4096], " ")]
}
