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
		Short: "add an item, account, OAuth2 client or cookie to the database",
		Long:  "option for the user to add accounts, items, OAuth2 clients and cookies to the database for the main usage",
	}

	cli.rootCmd.AddCommand(addCmd)
	addCmd.AddCommand(cli.getAddAccountCommand())
	addCmd.AddCommand(cli.getAddItemCommand())
	addCmd.AddCommand(cli.getAddOAuthClientCommand())
	addCmd.AddCommand(cli.getAddCookieCommand())
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
	accountCmd.Flags().StringVarP(&username, "user", "U", "", "username you want to add (required)")
	accountCmd.Flags().StringVarP(&password, "password", "P", "", "password of the user (required)")
	accountCmd.Flags().StringVarP(&url, "url", "u", "", "url for the association of the account (required)")
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
	oauthClientCmd := &cobra.Command{
		Use:   "oauth",
		Short: "adds an OAuth2 client to the database",
		Long:  "checks the passed url to assign the passed OAuth2 client to a module and save it to the database",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if clientID == "" && accessToken == "" {
				return fmt.Errorf("either clientID or accessToken is required as argument")
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			cli.watcher.AddOAuthClientByURI(url, clientID, clientSecret, accessToken, refreshToken)
		},
	}
	oauthClientCmd.Flags().StringVar(&clientID, "client-id", "", "OAuth2 client ID")
	oauthClientCmd.Flags().StringVar(&clientSecret, "client-secret", "", "OAuth2 client secret")
	oauthClientCmd.Flags().StringVar(&accessToken, "access-token", "", "OAuth2 access token")
	oauthClientCmd.Flags().StringVar(&refreshToken, "refresh-token", "", "OAuth2 refresh token")
	oauthClientCmd.Flags().StringVarP(&url, "url", "u", "", "url for the association of the OAuth2 client (required)")
	_ = oauthClientCmd.MarkFlagRequired("url")

	return oauthClientCmd
}

// getAddCookieCommand returns the command for the add cookie sub command
func (cli *CliApplication) getAddCookieCommand() *cobra.Command {
	var (
		url        string
		name       string
		value      string
		expiration string
	)

	// add the account option, requires username, password and uri
	cookieCmd := &cobra.Command{
		Use:   "cookie",
		Short: "adds a cookie to the database",
		Long:  "checks the passed url to assign the passed cookie to a module and save it to the database",
		Run: func(cmd *cobra.Command, args []string) {
			cli.watcher.AddCookieByURI(url, name, value, expiration)
		},
	}
	cookieCmd.Flags().StringVarP(&name, "name", "N", "", "cookie name (required)")
	cookieCmd.Flags().StringVarP(&value, "value", "V", "", "cookie value (required)")
	cookieCmd.Flags().StringVarP(&expiration, "expiration", "e", "", "cookie expiration")
	cookieCmd.Flags().StringVarP(&url, "url", "u", "", "url for the association of the cookie (required)")
	_ = cookieCmd.MarkFlagRequired("url")
	_ = cookieCmd.MarkFlagRequired("name")
	_ = cookieCmd.MarkFlagRequired("value")

	return cookieCmd
}
