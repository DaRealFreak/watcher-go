package watcher

import (
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	"github.com/DaRealFreak/watcher-go/pkg/update"
	"github.com/spf13/cobra"
)

// addListCommand adds the update sub command
func (cli *CliApplication) addUpdateCommand() {
	// general add option
	addCmd := &cobra.Command{
		Use:   "update",
		Short: "update the application or an item/account in the database",
		Long:  "option for the user to update the application or items/accounts in the database",
		Run: func(cmd *cobra.Command, args []string) {
			err := update.NewUpdateChecker().UpdateApplication()
			raven.CheckError(err)
		},
	}

	cli.rootCmd.AddCommand(addCmd)
	addCmd.AddCommand(cli.getUpdateAccountCommand())
	addCmd.AddCommand(cli.getUpdateItemCommand())
}

// getUpdateAccountCommand returns the command for the update account sub command
func (cli *CliApplication) getUpdateAccountCommand() *cobra.Command {
	var url string
	var user string
	var password string
	accountCmd := &cobra.Command{
		Use:   "account",
		Short: "updates the saved account",
		Long:  "updates the saved account in the database(new password, enable/disable)",
		Run: func(cmd *cobra.Command, args []string) {
			cli.watcher.UpdateAccountByURI(url, user, password)
		},
	}
	accountCmd.Flags().StringVar(&url, "url", "", "url of module (required)")
	accountCmd.Flags().StringVarP(&user, "user", "u", "", "username (required)")
	accountCmd.Flags().StringVarP(&password, "password", "p", "", "new password (required)")
	_ = accountCmd.MarkFlagRequired("url")
	_ = accountCmd.MarkFlagRequired("user")
	_ = accountCmd.MarkFlagRequired("password")
	accountCmd.AddCommand(cli.getEnableAccountCommand())
	accountCmd.AddCommand(cli.getDisableAccountCommand())
	return accountCmd
}

// getEnableAccountCommand returns the command for the update account enable sub command
func (cli *CliApplication) getEnableAccountCommand() *cobra.Command {
	var url string
	var user string

	enableCmd := &cobra.Command{
		Use:   "enable",
		Short: "enables an account based on the username",
		Long:  "update the database to set the user of the module to enabled",
		Run: func(cmd *cobra.Command, args []string) {
			cli.watcher.UpdateAccountDisabledStatusByURI(url, user, false)
		},
	}
	enableCmd.Flags().StringVar(&url, "url", "", "url of module (required)")
	enableCmd.Flags().StringVarP(&user, "user", "u", "", "username (required)")
	_ = enableCmd.MarkFlagRequired("url")
	_ = enableCmd.MarkFlagRequired("user")
	return enableCmd
}

// getDisableAccountCommand returns the command for the update account disable sub command
func (cli *CliApplication) getDisableAccountCommand() *cobra.Command {
	var url string
	var user string

	disableCmd := &cobra.Command{
		Use:   "disable",
		Short: "disable an account based on the username",
		Long:  "update the database to set the user of the module to disabled",
		Run: func(cmd *cobra.Command, args []string) {
			cli.watcher.UpdateAccountDisabledStatusByURI(url, user, true)
		},
	}
	disableCmd.Flags().StringVar(&url, "url", "", "url of module (required)")
	disableCmd.Flags().StringVarP(&user, "user", "u", "", "username (required)")
	_ = disableCmd.MarkFlagRequired("url")
	_ = disableCmd.MarkFlagRequired("user")
	return disableCmd
}

// getUpdateItemCommand returns the command for the update item sub command
func (cli *CliApplication) getUpdateItemCommand() *cobra.Command {
	var url string
	var current string

	itemCmd := &cobra.Command{
		Use:   "item",
		Short: "updates the saved current item",
		Long:  "updates the saved current item of an item in the database, creates an entry if it doesn't exist yet",
		Run: func(cmd *cobra.Command, args []string) {
			module := cli.watcher.ModuleFactory.GetModuleFromURI(url)
			trackedItem := cli.watcher.DbCon.GetFirstOrCreateTrackedItem(url, module)
			cli.watcher.DbCon.UpdateTrackedItem(trackedItem, current)
		},
	}
	itemCmd.Flags().StringVar(&url, "url", "", "url of tracked item you want to update (required)")
	itemCmd.Flags().StringVarP(&current, "current", "c", "", "current item in case you don't want to download older items")
	_ = itemCmd.MarkFlagRequired("url")
	_ = itemCmd.MarkFlagRequired("current")
	itemCmd.AddCommand(cli.getEnableItemCommand())
	itemCmd.AddCommand(cli.getDisableItemCommand())
	return itemCmd
}

// getEnableItemCommand returns the command for the update item enable sub command
func (cli *CliApplication) getEnableItemCommand() *cobra.Command {
	enableCmd := &cobra.Command{
		Use:   "enable [urls]",
		Short: "enables passed items based on the passed urls",
		Long:  "changes the completion status on the passed items to false, causing them to get checked again",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			for _, url := range args {
				module := cli.watcher.ModuleFactory.GetModuleFromURI(url)
				trackedItem := cli.watcher.DbCon.GetFirstOrCreateTrackedItem(url, module)
				cli.watcher.DbCon.ChangeTrackedItemCompleteStatus(trackedItem, false)
			}
		},
	}
	return enableCmd
}

// getDisableItemCommand returns the command for the update item disable sub command
func (cli *CliApplication) getDisableItemCommand() *cobra.Command {
	enableCmd := &cobra.Command{
		Use:   "disable [urls]",
		Short: "disables passed items based on the passed urls",
		Long:  "changes the completion status on the passed items to true, causing them to not get checked again",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			for _, url := range args {
				module := cli.watcher.ModuleFactory.GetModuleFromURI(url)
				trackedItem := cli.watcher.DbCon.GetFirstOrCreateTrackedItem(url, module)
				cli.watcher.DbCon.ChangeTrackedItemCompleteStatus(trackedItem, true)
			}
		},
	}
	return enableCmd
}
