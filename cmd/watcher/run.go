package watcher

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// addRunCommand adds the run sub command
func (cli *CliApplication) addRunCommand() {
	var downloadDirectory string
	var moduleURL string
	var runParallel bool
	// runs the main functionality to update all tracked items
	var runCmd = &cobra.Command{
		Use:   "run",
		Short: "update all tracked items",
		Run: func(cmd *cobra.Command, args []string) {
			cli.watcher.Run(moduleURL, runParallel)
		},
	}
	runCmd.Flags().StringVarP(
		&downloadDirectory,
		"directory",
		"d",
		"",
		"download directory (will be saved in config file)",
	)
	runCmd.Flags().StringVarP(
		&moduleURL,
		"url",
		"u",
		"",
		"url of module you want to run",
	)
	runCmd.Flags().BoolVarP(
		&runParallel,
		"parallel",
		"p",
		false,
		"run modules parallel",
	)
	_ = viper.BindPFlag("download.directory", runCmd.Flags().Lookup("directory"))
	cli.rootCmd.AddCommand(runCmd)
}
