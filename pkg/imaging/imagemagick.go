// Package imaging contains relevant general functions for imaging functions used by the sub packages
package imaging

import "runtime"

// GetImageMagickEnv retrieves the executable path and possible arguments for ImageMagick
// based on the OS it is running on
func GetImageMagickEnv(tool string) (executable string, args []string) {
	if runtime.GOOS == "windows" {
		executable = "magick"

		args = append(args, tool)
	} else {
		executable = tool
	}

	return
}
