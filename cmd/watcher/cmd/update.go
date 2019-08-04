package cmd

import "github.com/spf13/cobra"

func init() {
	// general add option
	addCmd := &cobra.Command{
		Use:   "update",
		Short: "update an item or account in the database",
		Long:  "option for the user to update accounts/items in the database",
	}

	RootCmd.AddCommand(addCmd)
	addCmd.AddCommand(getUpdateAccountCommand())
	addCmd.AddCommand(getUpdateItemCommand())
}

func getUpdateAccountCommand() *cobra.Command {
	var url string
	var user string
	var password string
	accountCmd := &cobra.Command{
		Use:   "account",
		Short: "updates the saved current item",
		Long:  "updates the saved current item of an item in the database",
		Run: func(cmd *cobra.Command, args []string) {
			WatcherApp.UpdateAccountByUri(url, user, password)
		},
	}
	accountCmd.Flags().StringVarP(&url, "url", "", "", "url of module (required)")
	accountCmd.Flags().StringVarP(&user, "user", "u", "", "username (required)")
	accountCmd.Flags().StringVarP(&password, "password", "p", "", "new password (required)")
	_ = accountCmd.MarkFlagRequired("url")
	_ = accountCmd.MarkFlagRequired("user")
	_ = accountCmd.MarkFlagRequired("password")
	accountCmd.AddCommand(getEnableAccountCommand())
	accountCmd.AddCommand(getDisableAccountCommand())
	return accountCmd
}

func getEnableAccountCommand() *cobra.Command {
	var url string
	var user string

	disableCmd := &cobra.Command{
		Use:   "enable",
		Short: "enables an account based on the username",
		Long:  "update the database to set the user of the module to enabled",
		Run: func(cmd *cobra.Command, args []string) {
			WatcherApp.UpdateAccountDisabledStatusByUri(url, user, false)
		},
	}
	disableCmd.Flags().StringVarP(&url, "url", "", "", "url of module (required)")
	disableCmd.Flags().StringVarP(&user, "user", "u", "", "username (required)")
	_ = disableCmd.MarkFlagRequired("url")
	_ = disableCmd.MarkFlagRequired("user")
	return disableCmd
}

func getDisableAccountCommand() *cobra.Command {
	var url string
	var user string

	disableCmd := &cobra.Command{
		Use:   "disable",
		Short: "disable an account based on the username",
		Long:  "update the database to set the user of the module to disabled",
		Run: func(cmd *cobra.Command, args []string) {
			WatcherApp.UpdateAccountDisabledStatusByUri(url, user, true)
		},
	}
	disableCmd.Flags().StringVarP(&url, "url", "", "", "url of module (required)")
	disableCmd.Flags().StringVarP(&user, "user", "u", "", "username (required)")
	_ = disableCmd.MarkFlagRequired("url")
	_ = disableCmd.MarkFlagRequired("user")
	return disableCmd
}

func getUpdateItemCommand() *cobra.Command {
	var url string
	var current string

	itemCmd := &cobra.Command{
		Use:   "item",
		Short: "updates the saved current item",
		Long:  "updates the saved current item of an item in the database, creates an entry if it doesn't exist yet",
		Run: func(cmd *cobra.Command, args []string) {
			module := WatcherApp.ModuleFactory.GetModuleFromUri(url)
			trackedItem := WatcherApp.DbCon.GetFirstOrCreateTrackedItem(url, module)
			WatcherApp.DbCon.UpdateTrackedItem(trackedItem, current)
		},
	}
	itemCmd.Flags().StringVarP(&url, "url", "", "", "url of item you want to track (required)")
	itemCmd.Flags().StringVarP(&current, "current", "c", "", "current item in case you don't want to download older items")
	_ = itemCmd.MarkFlagRequired("url")
	_ = itemCmd.MarkFlagRequired("current")
	return itemCmd
}
