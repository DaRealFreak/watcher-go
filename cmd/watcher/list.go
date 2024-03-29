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
		Long:  "option for the user to list all items, accounts, OAuth2 clients, cookies and modules from the database",
	}

	cli.rootCmd.AddCommand(listCmd)
	listCmd.AddCommand(cli.getListAccountsCommand())
	listCmd.AddCommand(cli.getListOAuthClientsCommand())
	listCmd.AddCommand(cli.getListCookiesCommand())
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
	accountCmd.Flags().StringVarP(&url, "url", "u", "", "url of module")

	return accountCmd
}

// getListOAuthClientsCommand returns the command for the list oauth sub command
func (cli *CliApplication) getListOAuthClientsCommand() *cobra.Command {
	var url string

	oauthClientsCmd := &cobra.Command{
		Use:   "oauth",
		Short: "displays all OAuth2 clients",
		Long:  "displays all OAuth2 clients currently in the database",
		Run: func(cmd *cobra.Command, args []string) {
			cli.watcher.ListOAuthClients(url)
		},
	}
	oauthClientsCmd.Flags().StringVarP(&url, "url", "u", "", "url of module")

	return oauthClientsCmd
}

func (cli *CliApplication) getListCookiesCommand() *cobra.Command {
	var url string

	cookiesCmd := &cobra.Command{
		Use:   "cookies",
		Short: "displays all cookies",
		Long:  "displays all cookies currently in the database",
		Run: func(cmd *cobra.Command, args []string) {
			cli.watcher.ListCookies(url)
		},
	}
	cookiesCmd.Flags().StringVarP(&url, "url", "u", "", "url of module")

	return cookiesCmd
}

// getListItemsCommand returns the command for the list items sub command
func (cli *CliApplication) getListItemsCommand() *cobra.Command {
	var (
		url              string
		partial          string
		includeCompleted bool
	)

	itemCmd := &cobra.Command{
		Use:   "items",
		Short: "displays all items",
		Long:  "displays all items currently in the database",
		Run: func(cmd *cobra.Command, args []string) {
			cli.watcher.ListTrackedItems(url, includeCompleted, partial)
		},
	}

	itemCmd.Flags().StringVarP(&url, "url", "u", "", "url of module")
	itemCmd.Flags().StringVarP(&partial, "partial", "p", "", "part of the item term")
	itemCmd.Flags().BoolVar(&includeCompleted, "include-completed", true, "should completed items be included in the list")

	return itemCmd
}

// getListAllCommand returns the command for the list all sub command
func (cli *CliApplication) getListAllCommand() *cobra.Command {
	var (
		url     string
		partial string
	)

	allCmd := &cobra.Command{
		Use:   "all",
		Short: "displays modules, accounts and items in the database",
		Long:  "displays all modules, accounts and items currently in the database",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Modules:")
			cli.watcher.ListRegisteredModules(url)
			fmt.Println("\n ")
			fmt.Println("Accounts:")
			cli.watcher.ListAccounts(url)
			fmt.Println("\n ")
			fmt.Println("OAuth2 Clients:")
			cli.watcher.ListOAuthClients(url)
			fmt.Println("\n ")
			fmt.Println("Cookies:")
			cli.watcher.ListCookies(url)
			fmt.Println("\n ")
			fmt.Println("Tracked Items:")
			cli.watcher.ListTrackedItems(url, true, partial)
		},
	}

	allCmd.Flags().StringVarP(&url, "url", "u", "", "url of module")
	allCmd.Flags().StringVarP(&partial, "partial", "p", "", "part of the item term")

	return allCmd
}

// getListModulesCommand returns the command for the list modules sub command
func (cli *CliApplication) getListModulesCommand() *cobra.Command {
	var url string

	modulesCmd := &cobra.Command{
		Use:   "modules",
		Short: "shows all registered modules",
		Long:  "shows all currently implemented and registered modules",
		Run: func(cmd *cobra.Command, args []string) {
			cli.watcher.ListRegisteredModules(url)
		},
	}

	modulesCmd.Flags().StringVarP(&url, "url", "u", "", "url of module")

	return modulesCmd
}
