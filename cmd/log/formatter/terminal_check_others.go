// +build !windows

package formatter

import (
	"io"
	"os"

	"github.com/mattn/go-isatty"
)

// isTerminal checks if we are currently in a terminal
func (f *Formatter) isTerminal(writer io.Writer) bool {
	// check the type since the file descriptor is only callable for files, so we can't access it directly
	switch out := writer.(type) {
	case *os.File:
		return isatty.IsCygwinTerminal(out.Fd())
	default:
		return false
	}
}
