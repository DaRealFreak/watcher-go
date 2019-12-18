package watcher

import (
	"fmt"

	"github.com/DaRealFreak/watcher-go/pkg/raven"
	"github.com/DaRealFreak/watcher-go/pkg/update"
	"github.com/spf13/cobra"
)

// addUpdateCommand adds the update sub command
func (cli *CliApplication) addUpdateCommand() {
	// general add option
	addCmd := &cobra.Command{
		Use:   "update",
		Short: "update the application or an item/account/OAuth2 client/cookie in the database",
		Long:  "option for the user to update the application or items/accounts/OAuth2 clients/cookies in the database",
		Run: func(cmd *cobra.Command, args []string) {
			err := update.NewUpdateChecker().UpdateApplication()
			raven.CheckError(err)
		},
	}

	cli.rootCmd.AddCommand(addCmd)
	addCmd.AddCommand(cli.getUpdateAccountCommand())
	addCmd.AddCommand(cli.getUpdateOAuthClientCommand())
	addCmd.AddCommand(cli.getUpdateItemCommand())
	addCmd.AddCommand(cli.getUpdateCookieCommand())
}

// getUpdateAccountCommand returns the command for the update account sub command
func (cli *CliApplication) getUpdateAccountCommand() *cobra.Command {
	var (
		url      string
		user     string
		password string
	)

	accountCmd := &cobra.Command{
		Use:   "account",
		Short: "updates the saved account",
		Long:  "updates the saved account in the database(new password, enable/disable)",
		Run: func(cmd *cobra.Command, args []string) {
			module := cli.watcher.ModuleFactory.GetModuleFromURI(url)
			cli.watcher.DbCon.UpdateAccount(user, password, module)
		},
	}

	accountCmd.Flags().StringVarP(&url, "url", "u", "", "url of module (required)")
	accountCmd.Flags().StringVarP(&user, "user", "U", "", "username (required)")
	accountCmd.Flags().StringVarP(&password, "password", "P", "", "new password (required)")

	_ = accountCmd.MarkFlagRequired("url")
	_ = accountCmd.MarkFlagRequired("user")
	_ = accountCmd.MarkFlagRequired("password")

	accountCmd.AddCommand(cli.getEnableAccountCommand())
	accountCmd.AddCommand(cli.getDisableAccountCommand())

	return accountCmd
}

// getEnableAccountCommand returns the command for the update account enable sub command
func (cli *CliApplication) getEnableAccountCommand() *cobra.Command {
	var (
		url  string
		user string
	)

	enableCmd := &cobra.Command{
		Use:   "enable",
		Short: "enables an account based on the username",
		Long:  "update the database to set the user of the module to enabled",
		Run: func(cmd *cobra.Command, args []string) {
			module := cli.watcher.ModuleFactory.GetModuleFromURI(url)
			cli.watcher.DbCon.UpdateAccountDisabledStatus(user, false, module)
		},
	}

	enableCmd.Flags().StringVarP(&url, "url", "u", "", "url of module (required)")
	enableCmd.Flags().StringVarP(&user, "user", "U", "", "username (required)")

	_ = enableCmd.MarkFlagRequired("url")
	_ = enableCmd.MarkFlagRequired("user")

	return enableCmd
}

// getDisableAccountCommand returns the command for the update account disable sub command
func (cli *CliApplication) getDisableAccountCommand() *cobra.Command {
	var (
		url  string
		user string
	)

	disableCmd := &cobra.Command{
		Use:   "disable",
		Short: "disable an account based on the username",
		Long:  "update the database to set the user of the module to disabled",
		Run: func(cmd *cobra.Command, args []string) {
			module := cli.watcher.ModuleFactory.GetModuleFromURI(url)
			cli.watcher.DbCon.UpdateAccountDisabledStatus(user, true, module)
		},
	}

	disableCmd.Flags().StringVarP(&url, "url", "u", "", "url of module (required)")
	disableCmd.Flags().StringVarP(&user, "user", "U", "", "username (required)")

	_ = disableCmd.MarkFlagRequired("url")
	_ = disableCmd.MarkFlagRequired("user")

	return disableCmd
}

// getUpdateAccountCommand returns the command for the update oauth sub command
func (cli *CliApplication) getUpdateOAuthClientCommand() *cobra.Command {
	var (
		url          string
		clientID     string
		clientSecret string
		accessToken  string
		refreshToken string
	)

	accountCmd := &cobra.Command{
		Use:   "oauth",
		Short: "updates the saved OAuth2 client",
		Long:  "updates the saved OAuth2 client in the database (new client secret/access token/refresh token)",
		Run: func(cmd *cobra.Command, args []string) {
			module := cli.watcher.ModuleFactory.GetModuleFromURI(url)
			cli.watcher.DbCon.UpdateOAuthClient(clientID, clientSecret, accessToken, refreshToken, module)
		},
	}

	accountCmd.Flags().StringVar(&clientID, "client-id", "", "OAuth2 client ID")
	accountCmd.Flags().StringVar(&clientSecret, "client-secret", "", "OAuth2 client secret")
	accountCmd.Flags().StringVar(&accessToken, "access-token", "", "OAuth2 access token")
	accountCmd.Flags().StringVar(&refreshToken, "refresh-token", "", "OAuth2 refresh token")

	accountCmd.Flags().StringVarP(&url, "url", "u", "", "url of module (required)")

	_ = accountCmd.MarkFlagRequired("url")
	_ = accountCmd.MarkFlagRequired("client-id")

	accountCmd.AddCommand(cli.getEnableOAuthClientCommand())
	accountCmd.AddCommand(cli.getDisableOAuthClientCommand())

	return accountCmd
}

// getEnableOAuthClientCommand returns the command for the update oauth enable sub command
// nolint: dupl
func (cli *CliApplication) getEnableOAuthClientCommand() *cobra.Command {
	var (
		url         string
		clientID    string
		accessToken string
	)

	disableCmd := &cobra.Command{
		Use:   "disable",
		Short: "disable an OAuth2 client based on the client ID or access token",
		Long:  "update the database to set the OAuth2 client of the module to disabled",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if clientID == "" && accessToken == "" {
				return fmt.Errorf("either clientID or accessToken is required as argument")
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			module := cli.watcher.ModuleFactory.GetModuleFromURI(url)
			cli.watcher.DbCon.UpdateOAuthClientDisabledStatus(clientID, accessToken, true, module)
		},
	}

	disableCmd.Flags().StringVarP(&url, "url", "u", "", "url of module (required)")
	disableCmd.Flags().StringVar(&clientID, "client-id", "", "OAuth2 client ID")
	disableCmd.Flags().StringVar(&accessToken, "access-token", "", "OAuth2 access token")

	_ = disableCmd.MarkFlagRequired("url")

	return disableCmd
}

// getDisableOAuthClientCommand returns the command for the update oauth disable sub command
// nolint: dupl
func (cli *CliApplication) getDisableOAuthClientCommand() *cobra.Command {
	var (
		url         string
		clientID    string
		accessToken string
	)

	enableCmd := &cobra.Command{
		Use:   "enable",
		Short: "enables an OAuth2 client based on the client ID or access token",
		Long:  "update the database to set the OAuth2 client of the module to enabled",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if clientID == "" && accessToken == "" {
				return fmt.Errorf("either clientID or accessToken is required as argument")
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			module := cli.watcher.ModuleFactory.GetModuleFromURI(url)
			cli.watcher.DbCon.UpdateOAuthClientDisabledStatus(clientID, accessToken, false, module)
		},
	}

	enableCmd.Flags().StringVarP(&url, "url", "u", "", "url of module (required)")
	enableCmd.Flags().StringVar(&clientID, "client-id", "", "OAuth2 client ID")
	enableCmd.Flags().StringVar(&accessToken, "access-token", "", "OAuth2 access token")

	_ = enableCmd.MarkFlagRequired("url")

	return enableCmd
}

// getUpdateItemCommand returns the command for the update item sub command
func (cli *CliApplication) getUpdateItemCommand() *cobra.Command {
	var (
		url     string
		current string
	)

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

	itemCmd.Flags().StringVarP(&url, "url", "u", "", "url of tracked item you want to update (required)")
	itemCmd.Flags().StringVarP(&current, "current", "c", "", "current item in case you don't want to download older items")

	_ = itemCmd.MarkFlagRequired("url")
	_ = itemCmd.MarkFlagRequired("current")

	itemCmd.AddCommand(cli.getEnableItemCommand())
	itemCmd.AddCommand(cli.getDisableItemCommand())

	return itemCmd
}

func (cli *CliApplication) getUpdateCookieCommand() *cobra.Command {
	var (
		url        string
		name       string
		value      string
		expiration string
	)

	// add the account option, requires username, password and uri
	cookieCmd := &cobra.Command{
		Use:   "cookie",
		Short: "updates the cookie for a new value and expiration date",
		Long:  "updates the saved cookie in the database, creates an entry if it doesn't exist yet",
		Run: func(cmd *cobra.Command, args []string) {
			module := cli.watcher.ModuleFactory.GetModuleFromURI(url)
			cli.watcher.DbCon.UpdateCookie(name, value, expiration, module)
		},
	}
	cookieCmd.Flags().StringVarP(&name, "name", "N", "", "cookie name (required)")
	cookieCmd.Flags().StringVarP(&value, "value", "V", "", "cookie value (required)")
	cookieCmd.Flags().StringVarP(&expiration, "expiration", "e", "", "cookie expiration")
	cookieCmd.Flags().StringVarP(&url, "url", "u", "", "url for the association of the cookie (required)")
	_ = cookieCmd.MarkFlagRequired("url")
	_ = cookieCmd.MarkFlagRequired("name")
	_ = cookieCmd.MarkFlagRequired("value")

	cookieCmd.AddCommand(cli.getEnableCookieCommand())
	cookieCmd.AddCommand(cli.getDisableCookieCommand())

	return cookieCmd
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

// getEnableCookieCommand returns the command for the update cookie enable sub command
func (cli *CliApplication) getEnableCookieCommand() *cobra.Command {
	var (
		url  string
		name string
	)

	enableCmd := &cobra.Command{
		Use:   "enable",
		Short: "enables the cookie matching to the passed module and name",
		Long:  "changes the disabled status on the matching cookie to false, causing the cookie to be used again",
		Run: func(cmd *cobra.Command, args []string) {
			module := cli.watcher.ModuleFactory.GetModuleFromURI(url)
			cli.watcher.DbCon.UpdateCookieDisabledStatus(name, false, module)
		},
	}

	enableCmd.Flags().StringVarP(&url, "url", "u", "", "url of module (required)")
	enableCmd.Flags().StringVarP(&name, "name", "N", "", "cookie name (required)")

	_ = enableCmd.MarkFlagRequired("url")
	_ = enableCmd.MarkFlagRequired("name")

	return enableCmd
}

// getDisableCookieCommand returns the command for the update cookie disable sub command
func (cli *CliApplication) getDisableCookieCommand() *cobra.Command {
	var (
		url  string
		name string
	)

	disableCmd := &cobra.Command{
		Use:   "disable",
		Short: "disables the cookie matching to the passed module and name",
		Long:  "changes the disabled status on the matching cookie to true, causing the cookie to not be used",
		Run: func(cmd *cobra.Command, args []string) {
			module := cli.watcher.ModuleFactory.GetModuleFromURI(url)
			cli.watcher.DbCon.UpdateCookieDisabledStatus(name, true, module)
		},
	}

	disableCmd.Flags().StringVarP(&url, "url", "u", "", "url of module (required)")
	disableCmd.Flags().StringVarP(&name, "name", "N", "", "cookie name (required)")

	_ = disableCmd.MarkFlagRequired("url")
	_ = disableCmd.MarkFlagRequired("name")

	return disableCmd
}
