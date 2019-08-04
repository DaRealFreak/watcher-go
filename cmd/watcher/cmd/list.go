package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

func init() {
	// general add option
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "lists items or accounts from the database",
		Long:  "option for the user to list all items/accounts from the database",
	}

	RootCmd.AddCommand(listCmd)
	listCmd.AddCommand(getListAllCommand())
	listCmd.AddCommand(getListAccountsCommand())
	listCmd.AddCommand(getListItemsCommand())
	listCmd.AddCommand(getListModulesCommand())
}

func getListAccountsCommand() *cobra.Command {
	var url string
	accountCmd := &cobra.Command{
		Use:   "accounts",
		Short: "displays all accounts",
		Long:  "displays all accounts currently in the database",
		Run: func(cmd *cobra.Command, args []string) {
			WatcherApp.ListAccounts(url)
		},
	}
	accountCmd.Flags().StringVar(&url, "url", "", "url of module")
	return accountCmd
}

func getListItemsCommand() *cobra.Command {
	var url string
	var includeCompleted bool

	itemCmd := &cobra.Command{
		Use:   "items",
		Short: "displays all items",
		Long:  "displays all items currently in the database",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(includeCompleted)
			WatcherApp.ListTrackedItems(url, includeCompleted)
		},
	}
	itemCmd.Flags().StringVar(&url, "url", "", "url of module")
	itemCmd.Flags().BoolVar(&includeCompleted, "include-completed", true, "should completed items be included in the list")
	return itemCmd
}

func getListAllCommand() *cobra.Command {
	allCmd := &cobra.Command{
		Use:   "all",
		Short: "displays accounts and items in the database",
		Long:  "displays all accounts and items currently in the database",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Modules:")
			WatcherApp.ListRegisteredModules()
			fmt.Println("\n ")
			fmt.Println("Accounts:")
			WatcherApp.ListAccounts("")
			fmt.Println("\n ")
			fmt.Println("Tracked Items:")
			WatcherApp.ListTrackedItems("", true)
		},
	}
	return allCmd
}

func getListModulesCommand() *cobra.Command {
	modulesCmd := &cobra.Command{
		Use:   "modules",
		Short: "shows all registered modules",
		Long:  "shows all currently implemented and registered modules",
		Run: func(cmd *cobra.Command, args []string) {
			WatcherApp.ListRegisteredModules()
		},
	}
	return modulesCmd
}
