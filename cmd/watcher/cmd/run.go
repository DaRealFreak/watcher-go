package cmd

import (
	"fmt"
	"github.com/kubernetes/klog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	RootCmd.AddCommand(getRunCommand())

}

func getRunCommand() *cobra.Command {
	var downloadDirectory string
	// runs the main functionality to update all tracked items
	var runCmd = &cobra.Command{
		Use:   "run",
		Short: "update all tracked items",
		Run: func(cmd *cobra.Command, args []string) {
			viper.WriteConfig()
			for _, item := range WatcherApp.DbCon.GetTrackedItems(nil) {
				module := WatcherApp.ModuleFactory.GetModule(item.Module)
				if !module.IsLoggedIn() {
					WatcherApp.LoginToModule(module)
				}
				klog.Info(fmt.Sprintf("parsing item %s (current id: %s)", item.Uri, item.CurrentItem))
				module.Parse(item)
			}
		},
	}
	runCmd.PersistentFlags().StringVarP(&downloadDirectory, "directory", "d", "", "Download Directory (required)")
	_ = viper.BindPFlag("DownloadDirectory", runCmd.PersistentFlags().Lookup("directory"))

	return runCmd
}
