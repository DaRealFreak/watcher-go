package cmd

import (
	"fmt"
	"github.com/DaRealFreak/watcher-go/pkg/update"
	watcherApp "github.com/DaRealFreak/watcher-go/pkg/watcher"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

var WatcherApp *watcherApp.Watcher

var RootCmd = &cobra.Command{
	Use:   "app",
	Short: "Watcher keeps track of all media items you want to track.",
	Long: "An application written in Go to keep track of items from multiple sources.\n" +
		"On every downloaded media file the current index will get updated so you'll never miss a tracked item",
}
var cfgFile string
var logLevel string

// add arguments for root command
func init() {
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./.watcher.yaml)")
	RootCmd.PersistentFlags().StringVarP(&logLevel, "verbosity", "v", log.InfoLevel.String(), "log level (debug, info, warn, error, fatal, panic")
}

// main cli functionality
func Execute() {
	// initialize logger as the very first
	initLogger()
	// check for available updates
	err := update.NewUpdateChecker().CheckForAvailableUpdates()
	if err != nil {
		log.Fatal(err)
	}
	// define the main application
	WatcherApp = watcherApp.NewWatcher()
	// parse all configurations before executing the main command
	cobra.OnInitialize(initConfig)
	// read in environment variables that match
	viper.AutomaticEnv()

	if err := RootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
	// close the database to prevent any dangling data
	WatcherApp.DbCon.CloseConnection()
}

// initialize the logger
func initLogger() {
	log.SetOutput(os.Stdout)
	lvl, err := log.ParseLevel(logLevel)
	if err != nil {
		log.Fatal("could not configure logger, exiting")
		os.Exit(1)
	}
	log.SetLevel(lvl)
}

// read config file
func initConfig() {
	// Don't forget to read config either from cfgFile or from home directory!
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		ensureDefaultConfigFile()
		// Search config in current directory with name ".watcher" (without extension).
		viper.AddConfigPath("./")
		viper.SetConfigName(".watcher")
	}

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Can't read config:", err)
		os.Exit(1)
	}
}

// ensure that the default config file exists in case no config file is defined
func ensureDefaultConfigFile() {
	if _, err := os.Stat("./.watcher.yaml"); os.IsNotExist(err) {
		_, _ = os.Create(".watcher.yaml")
	}
}
