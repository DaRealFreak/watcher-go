//go:build windows

package fp

import (
	"syscall"
)

// MoveFile uses syscall to move file (even across drives, which isn't allowed normally for windows)
func MoveFile(src string, dst string) error {
	from, _ := syscall.UTF16PtrFromString(src)
	to, _ := syscall.UTF16PtrFromString(dst)
	return syscall.MoveFile(from, to)
}
