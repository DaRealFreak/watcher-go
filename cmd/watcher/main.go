package watcher

import (
	"fmt"
	"github.com/DaRealFreak/watcher-go/pkg/update"
	"github.com/DaRealFreak/watcher-go/pkg/version"
	watcherApp "github.com/DaRealFreak/watcher-go/pkg/watcher"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

type configuration struct {
	configurationFile string
	logLevel          string
}

type CliApplication struct {
	watcher       *watcherApp.Watcher
	rootCmd       *cobra.Command
	configuration *configuration
}

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
	app.addGeneralArguments()
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

// add arguments for root command
func (cli *CliApplication) addGeneralArguments() {
	cli.rootCmd.PersistentFlags().StringVar(&cli.configuration.configurationFile, "config", "", "config file (default is ./.watcher.yaml)")
	cli.rootCmd.PersistentFlags().StringVarP(&cli.configuration.logLevel, "verbosity", "v", log.InfoLevel.String(), "log level (debug, info, warn, error, fatal, panic")
}

// main cli functionality
func (cli *CliApplication) Execute() {
	// check for available updates
	update.NewUpdateChecker().CheckForAvailableUpdates()

	if err := cli.rootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
	// close the database to prevent any dangling data
	cli.watcher.DbCon.CloseConnection()
}

// initialize everything the app needs
func (cli *CliApplication) initWatcher() {
	// initialize the logger
	cli.initLogger()

	// read and parse the configuration
	cli.initConfig()
}

// initialize the logger
func (cli *CliApplication) initLogger() {
	log.SetOutput(os.Stdout)
	lvl, err := log.ParseLevel(cli.configuration.logLevel)
	if err != nil {
		log.Fatal("could not configure logger, exiting")
		os.Exit(1)
	}
	log.SetLevel(lvl)
}

// read config file
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

// ensure that the default config file exists in case no config file is defined
func (cli *CliApplication) ensureDefaultConfigFile() {
	if _, err := os.Stat("./.watcher.yaml"); os.IsNotExist(err) {
		_, _ = os.Create(".watcher.yaml")
	}
}
