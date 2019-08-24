package watcher

import (
	"fmt"
	"os"

	"github.com/DaRealFreak/watcher-go/pkg/raven"
	"github.com/DaRealFreak/watcher-go/pkg/update"
	"github.com/DaRealFreak/watcher-go/pkg/version"
	watcherApp "github.com/DaRealFreak/watcher-go/pkg/watcher"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

// CliApplication contains the structure for the Watcher application for the CLI interface
type CliApplication struct {
	watcher *watcherApp.Watcher
	rootCmd *cobra.Command
	config  *watcherApp.AppConfiguration
}

// NewWatcherApplication returns the main command using cobra
func NewWatcherApplication() *CliApplication {
	app := &CliApplication{
		config: new(watcherApp.AppConfiguration),
		rootCmd: &cobra.Command{
			Use:   "watcher",
			Short: "Watcher keeps track of all media items you want to track.",
			Long: "An application written in Go to keep track of items from multiple sources.\n" +
				"On every downloaded media file the current index will get updated so you'll never miss a tracked item",
			Version: version.VERSION,
			Run: func(cmd *cobra.Command, args []string) {
				// display the help as the root command since we require a root command run function
				// the persistent flags won't get initialized
				// if we don't have a run function in the root command making them unusable
				_ = cmd.Help()
			},
		},
	}
	app.addPersistentFlags()
	app.addAddCommand()
	app.addListCommand()
	app.addRunCommand()
	app.addUpdateCommand()
	app.addBackupCommand()
	app.addGenerateAutoCompletionCommand()

	// read in environment variables that match
	viper.AutomaticEnv()
	// parse all configurations before executing the main command
	cobra.OnInitialize(app.initWatcher)
	return app
}

// addPersistentFlags adds persistent flags to root command
func (cli *CliApplication) addPersistentFlags() {
	cli.rootCmd.PersistentFlags().StringVar(
		&cli.config.ConfigurationFile,
		"config",
		"",
		"config file (default is ./.watcher.yaml)",
	)
	cli.rootCmd.PersistentFlags().StringVar(
		&cli.config.Database,
		"database",
		"",
		"database file (default is ./watcher.db)",
	)
	cli.rootCmd.PersistentFlags().StringVarP(
		&cli.config.LogLevel,
		"verbosity",
		"v",
		log.InfoLevel.String(),
		"log level (debug, info, warn, error, fatal, panic",
	)
	cli.rootCmd.PersistentFlags().BoolVar(
		&cli.config.EnableSentry,
		"enable-sentry",
		false,
		"use sentry to send usage statistics/errors to the developer",
	)
	cli.rootCmd.PersistentFlags().BoolVar(
		&cli.config.DisableSentry,
		"disable-sentry",
		false,
		"disable sentry and don't send usage statistics/errors to the developer",
	)
	cli.rootCmd.PersistentFlags().BoolVar(
		&cli.config.Cli.ForceColors,
		"log-force-colors",
		false,
		"enforces colored output even for non-tty terminals",
	)
	cli.rootCmd.PersistentFlags().BoolVar(
		&cli.config.Cli.ForceFormat,
		"log-force-format",
		false,
		"enforces formatted output even for non-tty terminals",
	)
	_ = viper.BindPFlag("Database.Path", cli.rootCmd.PersistentFlags().Lookup("database"))
}

// Execute executes the root command, entry point for the CLI application
func (cli *CliApplication) Execute() {
	// check for available updates
	update.NewUpdateChecker().CheckForAvailableUpdates()

	if err := cli.rootCmd.Execute(); err != nil {
		os.Exit(-1)
	}

	// if watcher got initialized by any command we close the database connection to prevent dangling data
	if cli.watcher != nil {
		cli.watcher.DbCon.CloseConnection()
	}
}

// initWatcher initializes everything the CLI application needs
func (cli *CliApplication) initWatcher() {
	// initialize the logger
	cli.initLogger()

	// read and parse the configuration
	cli.initConfig()

	// sentry toggle
	if cli.config.EnableSentry {
		viper.Set("watcher.sentry", true)
	}
	if cli.config.DisableSentry {
		viper.Set("watcher.sentry", false)
	}
	// setup sentry for error logging
	raven.SetupSentry()

	if viper.GetString("Database.Path") == "" {
		// if viper has no database path set up, set it to the default value
		viper.Set("Database.Path", "./watcher.db")
	}

	// initialize the watcher now after we parsed the configuration
	cli.watcher = watcherApp.NewWatcher()

	// save the configuration and check for errors
	err := viper.WriteConfig()
	raven.CheckError(err)
}

// initLogger initializes the logger
func (cli *CliApplication) initLogger() {
	log.SetOutput(os.Stdout)
	lvl, err := log.ParseLevel(cli.config.LogLevel)
	raven.CheckError(err)
	log.SetLevel(lvl)
	// set custom text formatter for the logger
	log.StandardLogger().Formatter = &prefixed.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
		ForceColors:     cli.config.Cli.ForceColors,
		ForceFormatting: cli.config.Cli.ForceFormat,
	}
}

// initConfig reads the set configuration file
func (cli *CliApplication) initConfig() {
	// Don't forget to read config either from cfgFile or from home directory!
	if cli.config.ConfigurationFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cli.config.ConfigurationFile)
	} else {
		cli.ensureDefaultConfigFile()
		// Search config in current directory with name ".watcher" (without extension).
		viper.AddConfigPath("./")
		viper.SetConfigName(".watcher")
	}

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Can't read config:", err)
		os.Exit(1)
	}
}

// ensureDefaultConfigFile ensures that the default config file exists in case no config file is defined
func (cli *CliApplication) ensureDefaultConfigFile() {
	if _, err := os.Stat("./.watcher.yaml"); os.IsNotExist(err) {
		_, _ = os.Create("./.watcher.yaml")
	}
}
