//go:build !windows

package fp

import (
	"os"
)

// MoveFile uses syscall to move file (even across drives, which isn't allowed normally for windows)
func MoveFile(src string, dst string) error {
	return os.Rename(src, dst)
}
