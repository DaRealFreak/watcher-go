package watcher

import (
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// addRunCommand adds the run sub command
func (cli *CliApplication) addRunCommand() {
	var downloadDirectory string
	var moduleURL string
	// runs the main functionality to update all tracked items
	var runCmd = &cobra.Command{
		Use:   "run",
		Short: "update all tracked items",
		Run: func(cmd *cobra.Command, args []string) {
			err := viper.WriteConfig()
			raven.CheckError(err)
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
	cli.rootCmd.AddCommand(runCmd)
}
