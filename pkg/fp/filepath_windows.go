//go:build windows

package fp

import (
	"syscall"
)

// MoveFile uses syscall to move file (even across drives, which isn't allowed normally for windows)
func MoveFile(src string, dst string) error {
	from, err := syscall.UTF16PtrFromString(src)
	if err != nil {
		return err
	}
	to, err := syscall.UTF16PtrFromString(dst)
	if err != nil {
		return err
	}
	return syscall.MoveFile(from, to)
}
