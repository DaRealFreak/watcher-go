package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	watcherApp "watcher-go/pkg/watcher"
)

var WatcherApp *watcherApp.Watcher

var RootCmd = &cobra.Command{
	Use:   "app",
	Short: "Watcher keeps track of all media items you want to track.",
	Long: "An application written in Go to keep track of items from multiple sources.\n" +
		"On every downloaded media file the current index will get updated so you'll never miss a tracked item",
}
var cfgFile string

// add arguments for root command
func init() {
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.watcher.yaml)")
}

// main cli functionality
func Execute() {
	WatcherApp = watcherApp.NewWatcher()
	cobra.OnInitialize(initConfig)
	viper.AutomaticEnv() // read in environment variables that match

	if err := RootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
	WatcherApp.DbCon.CloseConnection()
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
