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
		Force             bool
		ResetProgress     bool
		RunParallel       bool
		Items             []string
		DownloadDirectory string
		ModuleURL         []string
		DisableURL        []string
		// ProxyConnectionLimits caps simultaneous in-flight HTTP requests per
		// (proxy username, host eTLD+1) pool. A list (not a map) is used so
		// domain names containing dots survive Viper's nested-key flattening.
		// Example YAML:
		//   run:
		//     proxy_connection_limits:
		//       - domain: nordvpn.com
		//         max: 10
		//         cooldown_seconds: 10
		//       - domain: mullvad.net
		//         max: 5
		// Domains absent from the list are unlimited.
		ProxyConnectionLimits []ProxyConnectionLimit `mapstructure:"proxy_connection_limits"`
	}
}

// ProxyConnectionLimit defines a per-service connection cap. Used by the
// global ConnectionBudget to enforce per-(username, domain) pool limits.
// CooldownSeconds, if >0, delays a freshly-released slot from being handed
// to a *different* host within the same pool — gives the VPN's server-side
// connection counter time to decrement before the next host hits the cap.
type ProxyConnectionLimit struct {
	Domain          string `mapstructure:"domain"`
	Max             int    `mapstructure:"max"`
	CooldownSeconds int    `mapstructure:"cooldown_seconds,omitempty"`
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
