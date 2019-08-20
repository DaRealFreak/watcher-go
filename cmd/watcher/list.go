package watcher

import (
	"fmt"

	"github.com/spf13/cobra"
)

// addListCommand adds the list sub command
func (cli *CliApplication) addListCommand() {
	// general add option
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "lists items or accounts from the database",
		Long:  "option for the user to list all items/accounts from the database",
	}

	cli.rootCmd.AddCommand(listCmd)
	listCmd.AddCommand(cli.getListAccountsCommand())
	listCmd.AddCommand(cli.getListItemsCommand())
	listCmd.AddCommand(cli.getListModulesCommand())
	listCmd.AddCommand(cli.getListAllCommand())
}

// getListAccountsCommand returns the command for the list accounts sub command
func (cli *CliApplication) getListAccountsCommand() *cobra.Command {
	var url string
	accountCmd := &cobra.Command{
		Use:   "accounts",
		Short: "displays all accounts",
		Long:  "displays all accounts currently in the database",
		Run: func(cmd *cobra.Command, args []string) {
			cli.watcher.ListAccounts(url)
		},
	}
	accountCmd.Flags().StringVar(&url, "url", "", "url of module")
	return accountCmd
}

// getListItemsCommand returns the command for the list items sub command
func (cli *CliApplication) getListItemsCommand() *cobra.Command {
	var url string
	var includeCompleted bool

	itemCmd := &cobra.Command{
		Use:   "items",
		Short: "displays all items",
		Long:  "displays all items currently in the database",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(includeCompleted)
			cli.watcher.ListTrackedItems(url, includeCompleted)
		},
	}
	itemCmd.Flags().StringVar(&url, "url", "", "url of module")
	itemCmd.Flags().BoolVar(&includeCompleted, "include-completed", true, "should completed items be included in the list")
	return itemCmd
}

// getListAllCommand returns the command for the list all sub command
func (cli *CliApplication) getListAllCommand() *cobra.Command {
	allCmd := &cobra.Command{
		Use:   "all",
		Short: "displays modules, accounts and items in the database",
		Long:  "displays all modules, accounts and items currently in the database",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Modules:")
			cli.watcher.ListRegisteredModules()
			fmt.Println("\n ")
			fmt.Println("Accounts:")
			cli.watcher.ListAccounts("")
			fmt.Println("\n ")
			fmt.Println("Tracked Items:")
			cli.watcher.ListTrackedItems("", true)
		},
	}
	return allCmd
}

// getListModulesCommand returns the command for the list modules sub command
func (cli *CliApplication) getListModulesCommand() *cobra.Command {
	modulesCmd := &cobra.Command{
		Use:   "modules",
		Short: "shows all registered modules",
		Long:  "shows all currently implemented and registered modules",
		Run: func(cmd *cobra.Command, args []string) {
			cli.watcher.ListRegisteredModules()
		},
	}
	return modulesCmd
}
