package watcher

import (
	"github.com/DaRealFreak/watcher-go/pkg/update"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func (cli *CliApplication) addUpdateCommand() {
	// general add option
	addCmd := &cobra.Command{
		Use:   "update",
		Short: "update the application or an item/account in the database",
		Long:  "option for the user to update the application or items/accounts in the database",
		Run: func(cmd *cobra.Command, args []string) {
			err := update.NewUpdateChecker().UpdateApplication()
			if err != nil {
				log.Error(err)
			}
		},
	}

	cli.rootCmd.AddCommand(addCmd)
	addCmd.AddCommand(cli.getUpdateAccountCommand())
	addCmd.AddCommand(cli.getUpdateItemCommand())
}

func (cli *CliApplication) getUpdateAccountCommand() *cobra.Command {
	var url string
	var user string
	var password string
	accountCmd := &cobra.Command{
		Use:   "account",
		Short: "updates the saved current item",
		Long:  "updates the saved current item of an item in the database",
		Run: func(cmd *cobra.Command, args []string) {
			cli.watcher.UpdateAccountByUri(url, user, password)
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

func (cli *CliApplication) getEnableAccountCommand() *cobra.Command {
	var url string
	var user string

	disableCmd := &cobra.Command{
		Use:   "enable",
		Short: "enables an account based on the username",
		Long:  "update the database to set the user of the module to enabled",
		Run: func(cmd *cobra.Command, args []string) {
			cli.watcher.UpdateAccountDisabledStatusByUri(url, user, false)
		},
	}
	disableCmd.Flags().StringVar(&url, "url", "", "url of module (required)")
	disableCmd.Flags().StringVarP(&user, "user", "u", "", "username (required)")
	_ = disableCmd.MarkFlagRequired("url")
	_ = disableCmd.MarkFlagRequired("user")
	return disableCmd
}

func (cli *CliApplication) getDisableAccountCommand() *cobra.Command {
	var url string
	var user string

	disableCmd := &cobra.Command{
		Use:   "disable",
		Short: "disable an account based on the username",
		Long:  "update the database to set the user of the module to disabled",
		Run: func(cmd *cobra.Command, args []string) {
			cli.watcher.UpdateAccountDisabledStatusByUri(url, user, true)
		},
	}
	disableCmd.Flags().StringVar(&url, "url", "", "url of module (required)")
	disableCmd.Flags().StringVarP(&user, "user", "u", "", "username (required)")
	_ = disableCmd.MarkFlagRequired("url")
	_ = disableCmd.MarkFlagRequired("user")
	return disableCmd
}

func (cli *CliApplication) getUpdateItemCommand() *cobra.Command {
	var url string
	var current string

	itemCmd := &cobra.Command{
		Use:   "item",
		Short: "updates the saved current item",
		Long:  "updates the saved current item of an item in the database, creates an entry if it doesn't exist yet",
		Run: func(cmd *cobra.Command, args []string) {
			module := cli.watcher.ModuleFactory.GetModuleFromUri(url)
			trackedItem := cli.watcher.DbCon.GetFirstOrCreateTrackedItem(url, module)
			cli.watcher.DbCon.UpdateTrackedItem(trackedItem, current)
		},
	}
	itemCmd.Flags().StringVar(&url, "url", "", "url of tracked item you want to update (required)")
	itemCmd.Flags().StringVarP(&current, "current", "c", "", "current item in case you don't want to download older items")
	_ = itemCmd.MarkFlagRequired("url")
	_ = itemCmd.MarkFlagRequired("current")
	return itemCmd
}
