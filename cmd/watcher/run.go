package watcher

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func (cli *CliApplication) addRunCommand() {
	cli.rootCmd.AddCommand(cli.getRunCommand())
}

// retrieve the run command
func (cli *CliApplication) getRunCommand() *cobra.Command {
	var downloadDirectory string
	var moduleURL string
	// runs the main functionality to update all tracked items
	var runCmd = &cobra.Command{
		Use:   "run",
		Short: "update all tracked items",
		Run: func(cmd *cobra.Command, args []string) {
			err := viper.WriteConfig()
			if err != nil {
				log.Error("could not save the configuration")
			}
			cli.watcher.Run(moduleURL)
		},
	}
	runCmd.PersistentFlags().StringVarP(
		&downloadDirectory,
		"directory",
		"d",
		"",
		"download directory (will be saved in config file)",
	)
	runCmd.PersistentFlags().StringVarP(
		&moduleURL,
		"url",
		"u",
		"",
		"url of module you want to run",
	)
	_ = viper.BindPFlag("DownloadDirectory", runCmd.PersistentFlags().Lookup("directory"))
	return runCmd
}
