package watcher

import (
	"fmt"

	"github.com/spf13/cobra"
)

// addAddCommand adds the add sub command
func (cli *CliApplication) addAddCommand() {
	// general add option
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "add an item or account to the database",
		Long:  "option for the user to add accounts/items to the database for the main usage",
	}

	cli.rootCmd.AddCommand(addCmd)
	addCmd.AddCommand(cli.getAddAccountCommand())
	addCmd.AddCommand(cli.getAddItemCommand())
	addCmd.AddCommand(cli.getAddOAuthClientCommand())
}

// getAddItemCommand returns the command for the add item sub command
func (cli *CliApplication) getAddItemCommand() *cobra.Command {
	// add the item option, requires only the uri
	itemCmd := &cobra.Command{
		Use:   "item [urls]",
		Short: "adds an item to the database",
		Long:  "parses and adds the passed url into the tracked items if not tracked already",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			for _, url := range args {
				cli.watcher.AddItemByURI(url, "")
			}
		},
	}

	return itemCmd
}

// getAddAccountCommand returns the command for the add account sub command
func (cli *CliApplication) getAddAccountCommand() *cobra.Command {
	var (
		url      string
		username string
		password string
	)

	// add the account option, requires username, password and uri
	accountCmd := &cobra.Command{
		Use:   "account",
		Short: "adds an account to the database",
		Long:  "checks the passed url to assign the passed account/password to a module and save it to the database",
		Run: func(cmd *cobra.Command, args []string) {
			cli.watcher.AddAccountByURI(url, username, password)
		},
	}
	accountCmd.Flags().StringVarP(&username, "user", "u", "", "username you want to add (required)")
	accountCmd.Flags().StringVarP(&password, "password", "p", "", "password of the user (required)")
	accountCmd.Flags().StringVar(&url, "url", "", "url for the association of the account (required)")
	_ = accountCmd.MarkFlagRequired("user")
	_ = accountCmd.MarkFlagRequired("password")
	_ = accountCmd.MarkFlagRequired("url")

	return accountCmd
}

// getAddOAuthClientCommand returns the command for the add oauth sub command
func (cli *CliApplication) getAddOAuthClientCommand() *cobra.Command {
	var (
		url          string
		clientID     string
		clientSecret string
		accessToken  string
		refreshToken string
	)

	// add the account option, requires username, password and uri
	accountCmd := &cobra.Command{
		Use:   "oauth",
		Short: "adds an OAuth2 client to the database",
		Long:  "checks the passed url to assign the passed OAuth2 client to a module and save it to the database",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if clientID == "" && refreshToken == "" {
				return fmt.Errorf("either clientID or accessToken is required as argument")
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			cli.watcher.AddOAuthClientByURI(url, clientID, clientSecret, accessToken, refreshToken)
		},
	}
	accountCmd.Flags().StringVar(&clientID, "client-id", "", "OAuth2 client ID")
	accountCmd.Flags().StringVar(&clientSecret, "client-secret", "", "OAuth2 client secret")
	accountCmd.Flags().StringVar(&accessToken, "access-token", "", "OAuth2 access token")
	accountCmd.Flags().StringVar(&refreshToken, "refresh-token", "", "OAuth2 refresh token")
	accountCmd.Flags().StringVar(&url, "url", "", "url for the association of the OAuth2 client (required)")
	_ = accountCmd.MarkFlagRequired("url")

	return accountCmd
}
