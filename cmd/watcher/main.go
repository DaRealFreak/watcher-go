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
)

// configuration contains the persistent configurations across all commands
type configuration struct {
	configurationFile string
	logLevel          string
	enableSentry      bool
	disableSentry     bool
}

// CliApplication contains the structure for the Watcher application for the CLI interface
type CliApplication struct {
	watcher       *watcherApp.Watcher
	rootCmd       *cobra.Command
	configuration *configuration
}

// NewWatcherApplication returns the main command using cobra
func NewWatcherApplication() *CliApplication {
	app := &CliApplication{
		watcher: watcherApp.NewWatcher(),
		configuration: &configuration{
			configurationFile: "",
			logLevel:          "",
		},
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
		&cli.configuration.configurationFile,
		"config",
		"",
		"config file (default is ./.watcher.yaml)",
	)
	cli.rootCmd.PersistentFlags().StringVarP(
		&cli.configuration.logLevel,
		"verbosity",
		"v",
		log.InfoLevel.String(),
		"log level (debug, info, warn, error, fatal, panic",
	)
	cli.rootCmd.PersistentFlags().BoolVar(
		&cli.configuration.enableSentry,
		"enable-sentry",
		false,
		"use sentry to send usage statistics/errors to the developer",
	)
	cli.rootCmd.PersistentFlags().BoolVar(
		&cli.configuration.disableSentry,
		"disable-sentry",
		false,
		"disable sentry and don't send usage statistics/errors to the developer",
	)
}

// Execute executes the root command, entry point for the CLI application
func (cli *CliApplication) Execute() {
	// check for available updates
	update.NewUpdateChecker().CheckForAvailableUpdates()

	if err := cli.rootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
	// close the database to prevent any dangling data
	cli.watcher.DbCon.CloseConnection()
}

// initWatcher initializes everything the CLI application needs
func (cli *CliApplication) initWatcher() {
	// initialize the logger
	cli.initLogger()

	// read and parse the configuration
	cli.initConfig()

	// sentry toggle
	if cli.configuration.enableSentry {
		viper.Set("watcher.sentry", true)
	}
	if cli.configuration.disableSentry {
		viper.Set("watcher.sentry", false)
	}
	// setup sentry for error logging
	raven.SetupSentry()

	// save the configuration and check for errors
	err := viper.WriteConfig()
	raven.CheckError(err)
}

// initLogger initializes the logger
func (cli *CliApplication) initLogger() {
	log.SetOutput(os.Stdout)
	lvl, err := log.ParseLevel(cli.configuration.logLevel)
	raven.CheckError(err)
	log.SetLevel(lvl)
}

// initConfig reads the set configuration file
func (cli *CliApplication) initConfig() {
	// Don't forget to read config either from cfgFile or from home directory!
	if cli.configuration.configurationFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cli.configuration.configurationFile)
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
		_, _ = os.Create(".watcher.yaml")
	}
}
