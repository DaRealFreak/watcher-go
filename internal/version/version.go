// Package version contains the version and repository URL of the application
package version

var version string

// GetVersion returns the build version of the application
func GetVersion() string {
	// development build could have no version build flag set
	if version == "" {
		version = "99.99.99-dev"
	}

	return version
}
