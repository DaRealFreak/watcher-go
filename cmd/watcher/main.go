// Package watcher contains all configuration options for the command line interface
package watcher

import (
	"fmt"
	"os"
	"time"

	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/internal/configuration"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/DaRealFreak/watcher-go/internal/update"
	"github.com/DaRealFreak/watcher-go/internal/version"
	watcherApp "github.com/DaRealFreak/watcher-go/internal/watcher"
	"github.com/mattn/go-colorable"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// CliApplication contains the structure for the Watcher application for the CLI interface
type CliApplication struct {
	watcher *watcherApp.Watcher
	rootCmd *cobra.Command
	config  *configuration.AppConfiguration
}

// NewWatcherApplication returns the main command using cobra
func NewWatcherApplication() *CliApplication {
	app := &CliApplication{
		config: new(configuration.AppConfiguration),
		rootCmd: &cobra.Command{
			Use:   "watcher",
			Short: "Watcher keeps track of all media items you want to track.",
			Long: "An application written in Go to keep track of items from multiple sources.\n" +
				"On every downloaded media file the current index will get updated so you'll never miss a tracked item",
			Version: version.GetVersion(),
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
	app.addRestoreCommand()
	app.addModulesCommand()
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
		"config", "",
		"config file (default is "+watcherApp.DefaultConfigurationPath+")",
	)
	cli.rootCmd.PersistentFlags().StringVar(
		&cli.config.Database,
		"database", "",
		"database file (default is "+watcherApp.DefaultDatabasePath+")",
	)
	cli.rootCmd.PersistentFlags().StringVarP(
		&cli.config.LogLevel,
		"verbosity", "v", log.InfoLevel.String(),
		"log level (debug, info, warn, error, fatal, panic",
	)
	cli.rootCmd.PersistentFlags().BoolVar(
		&cli.config.EnableSentry,
		"enable-sentry", false,
		"use sentry to send usage statistics/errors to the developer",
	)
	cli.rootCmd.PersistentFlags().BoolVar(
		&cli.config.DisableSentry,
		"disable-sentry", false,
		"disable sentry and don't send usage statistics/errors to the developer",
	)
	cli.rootCmd.PersistentFlags().BoolVar(
		&cli.config.Cli.DisableColors,
		"log-disable-colors", false,
		"disables colors even for tty terminals",
	)
	cli.rootCmd.PersistentFlags().BoolVar(
		&cli.config.Cli.ForceColors,
		"log-force-colors", false,
		"enforces colored output even for non-tty terminals",
	)
	cli.rootCmd.PersistentFlags().BoolVar(
		&cli.config.Cli.DisableTimestamp,
		"log-disable-timestamp", false,
		"removes the time info of the log entries, useful if output is logged with a timestamp already",
	)
	cli.rootCmd.PersistentFlags().BoolVar(
		&cli.config.Cli.UseUppercaseLevel,
		"log-level-uppercase", false,
		"transforms the log levels into upper case",
	)
	cli.rootCmd.PersistentFlags().BoolVar(
		&cli.config.Cli.UseTimePassedAsTimestamp,
		"log-timestamp-passed-time", false,
		"uses the passed time since the program is running in seconds instead of a formatted time",
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
		viper.Set("Database.Path", watcherApp.DefaultDatabasePath)
	}

	// initialize the watcher now after we parsed the configuration
	cli.watcher = watcherApp.NewWatcher(cli.config)

	// save the configuration and check for errors
	raven.CheckError(viper.WriteConfig())
}

// initLogger initializes the logger
func (cli *CliApplication) initLogger() {
	log.SetOutput(colorable.NewColorableStdout())
	lvl, err := log.ParseLevel(cli.config.LogLevel)
	raven.CheckError(err)
	log.SetLevel(lvl)
	// set custom text formatter for the logger
	log.StandardLogger().Formatter = &formatter.Formatter{
		DisableColors:            cli.config.Cli.DisableColors,
		ForceColors:              cli.config.Cli.ForceColors,
		DisableTimestamp:         cli.config.Cli.DisableTimestamp,
		UseUppercaseLevel:        cli.config.Cli.UseUppercaseLevel,
		UseTimePassedAsTimestamp: cli.config.Cli.UseTimePassedAsTimestamp,
		TimestampFormat:          time.StampMilli,
		PadAllLogEntries:         true,
	}
}

// initConfig reads the set configuration file
func (cli *CliApplication) initConfig() {
	if cli.config.ConfigurationFile == "" {
		cli.config.ConfigurationFile = watcherApp.DefaultConfigurationPath
	}

	cli.ensureConfigurationFile()
	viper.SetConfigFile(cli.config.ConfigurationFile)

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Can't read config:", err)
		os.Exit(1)
	}
}

// ensureConfigurationFile ensures that the default config file exists in case no config file is defined
func (cli *CliApplication) ensureConfigurationFile() {
	if _, err := os.Stat(cli.config.ConfigurationFile); os.IsNotExist(err) {
		_, _ = os.Create(cli.config.ConfigurationFile)
	}
}
