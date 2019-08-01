package cmd

import (
	"github.com/kubernetes/klog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	RootCmd.AddCommand(getRunCommand())
}

// retrieve the run command
func getRunCommand() *cobra.Command {
	var downloadDirectory string
	// runs the main functionality to update all tracked items
	var runCmd = &cobra.Command{
		Use:   "run",
		Short: "update all tracked items",
		Run: func(cmd *cobra.Command, args []string) {
			err := viper.WriteConfig()
			if err != nil {
				klog.Error("Could not save the configuration")
			}

			WatcherApp.Run()
		},
	}
	runCmd.PersistentFlags().StringVarP(&downloadDirectory, "directory", "d", "", "Download Directory (required)")
	_ = viper.BindPFlag("DownloadDirectory", runCmd.PersistentFlags().Lookup("directory"))
	return runCmd
}
