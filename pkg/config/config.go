package config

// AppConfiguration contains the persistent configurations/settings across all commands
type AppConfiguration struct {
	ConfigurationFile string
	LogLevel          string
	EnableSentry      bool
	DisableSentry     bool
	// backup options
	Backup struct {
		Zip  bool
		Tar  bool
		Gzip bool
		SQL  bool
	}
	// cli specific options
	Cli struct {
		ForceColors bool
		ForceFormat bool
	}
}

// nolint:gochecknoglobals
var (
	// GlobalConfig contains all possible configurations to be used by the watcher application
	GlobalConfig = NewAppConfig()
)

// NewAppConfig generates a default app configuration
func NewAppConfig() *AppConfiguration {
	return new(AppConfiguration)
}
