package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	// general add option
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "add an item or account to the database",
		Long:  "option for the user to add accounts/items to the database for the main usage",
	}

	RootCmd.AddCommand(addCmd)
	addCmd.AddCommand(getAccountCommand())
	addCmd.AddCommand(getItemCommand())
}

// retrieve the command for add item
func getItemCommand() *cobra.Command {
	var url string
	var current string

	// add the item option, requires only the uri
	itemCmd := &cobra.Command{
		Use:   "item",
		Short: "adds an item to the database",
		Long:  "parses and adds the passed url into the tracked items if not already tracked",
		Run: func(cmd *cobra.Command, args []string) {
			WatcherApp.AddItemByUri(url, current)
		},
	}
	itemCmd.Flags().StringVarP(&url, "url", "", "", "url of item you want to track (required)")
	itemCmd.Flags().StringVarP(&current, "current", "", "", "current item in case you don't want to download older items")
	_ = itemCmd.MarkFlagRequired("url")
	return itemCmd
}

// retrieve the command for add account
func getAccountCommand() *cobra.Command {
	var url string
	var username string
	var password string

	// add the account option, requires username, password and uri
	accountCmd := &cobra.Command{
		Use:   "account",
		Short: "adds an account to the database",
		Long:  "checks the passed url to assign the passed account/password to a module and save it to the database",
		Run: func(cmd *cobra.Command, args []string) {
			WatcherApp.AddAccountByUri(url, username, password)
		},
	}
	accountCmd.Flags().StringVarP(&username, "username", "u", "", "username you want to add (required)")
	accountCmd.Flags().StringVarP(&password, "password", "p", "", "password of the user (required)")
	accountCmd.Flags().StringVarP(&url, "url", "", "", "url for the association of the account (required)")
	_ = accountCmd.MarkFlagRequired("username")
	_ = accountCmd.MarkFlagRequired("password")
	_ = accountCmd.MarkFlagRequired("url")
	return accountCmd
}
