package configuration

// AppConfiguration contains the persistent configurations/settings across all commands
type AppConfiguration struct {
	ConfigurationFile string
	LogLevel          string
	// database file location
	Database string
	// backup options
	Backup struct {
		BackupSettings
		Archive struct {
			Zip  bool
			Tar  bool
			Gzip bool
		}
	}
	Restore struct {
		BackupSettings
	}
	// cli specific options
	Cli struct {
		DisableColors            bool
		ForceColors              bool
		DisableTimestamp         bool
		UseUppercaseLevel        bool
		UseTimePassedAsTimestamp bool
	}
	// sentry toggles
	EnableSentry  bool
	DisableSentry bool
	// run specific options
	Run struct {
		ForceNew          bool
		RunParallel       bool
		Items             []string
		DownloadDirectory string
		ModuleURL         []string
		DisableURL        []string
	}
}

// BackupSettings are the possible configuration settings for backups and recoveries
type BackupSettings struct {
	Database struct {
		Accounts struct {
			Enabled bool
		}
		Items struct {
			Enabled bool
		}
		OAuth2Clients struct {
			Enabled bool
		}
		Cookies struct {
			Enabled bool
		}
		SQL bool
	}
	Settings bool
}
