package watcher

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// addRunCommand adds the run sub command
func (cli *CliApplication) addRunCommand() {
	// runs the main functionality to update all tracked items
	runCmd := &cobra.Command{
		Use:   "run [items]",
		Short: "update all tracked items or directly passed items",
		Long: "update all tracked items if no direct items are passed.\n" +
			"If items are directly passed only these will be updated.",
		Run: func(cmd *cobra.Command, args []string) {
			cli.config.Run.Items = args
			cli.watcher.Run()
		},
	}

	runCmd.Flags().StringVarP(
		&cli.config.Run.DownloadDirectory,
		"directory", "d", "",
		"download directory (will be saved in config file)",
	)
	runCmd.Flags().StringSliceVarP(
		&cli.config.Run.ModuleURL,
		"url", "u", []string{},
		"url of module you want to run",
	)
	runCmd.Flags().StringSliceVarP(
		&cli.config.Run.DisableURL,
		"disable", "x", []string{},
		"url of module you want don't want to run",
	)
	runCmd.Flags().BoolVarP(
		&cli.config.Run.RunParallel,
		"parallel", "p", false,
		"run modules parallel",
	)

	_ = viper.BindPFlag("download.directory", runCmd.Flags().Lookup("directory"))

	cli.rootCmd.AddCommand(runCmd)
}
