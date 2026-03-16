package watcher

import (
	"github.com/spf13/cobra"
)

// addImportCommand adds the import sub command
func (cli *CliApplication) addImportCommand() {
	importCmd := &cobra.Command{
		Use:   "import",
		Short: "import data from files into the database",
		Long:  "option for the user to import cookies or other data from files into the database",
	}

	cli.rootCmd.AddCommand(importCmd)
	importCmd.AddCommand(cli.getImportCookiesCommand())
	importCmd.AddCommand(cli.getImportCookiesClipboardCommand())
}

// getImportCookiesCommand returns the command for the import cookies sub command
func (cli *CliApplication) getImportCookiesCommand() *cobra.Command {
	var url string

	cookiesCmd := &cobra.Command{
		Use:   "cookies [file]",
		Short: "imports cookies from a Netscape cookie file",
		Long: "parses a Netscape/Mozilla format cookie file and imports all cookies into the database.\n" +
			"The module is auto-detected from the cookie domain. Use --url to override.",
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cli.watcher.ImportCookiesByURI(args[0], url)
		},
	}
	cookiesCmd.Flags().StringVarP(&url, "url", "u", "", "url to override module detection (optional)")

	return cookiesCmd
}

// getImportCookiesClipboardCommand returns the command for importing cookies from clipboard
func (cli *CliApplication) getImportCookiesClipboardCommand() *cobra.Command {
	var url string

	clipboardCmd := &cobra.Command{
		Use:   "cookies-clipboard",
		Short: "imports cookies from clipboard (Netscape format)",
		Long: "reads Netscape/Mozilla format cookie data from the clipboard and imports all cookies into the database.\n" +
			"Works with browser extensions like \"Get cookies.txt LOCALLY\" that copy cookies to clipboard.\n" +
			"The module is auto-detected from the cookie domain. Use --url to override.",
		Run: func(cmd *cobra.Command, args []string) {
			cli.watcher.ImportCookiesFromClipboardByURI(url)
		},
	}
	clipboardCmd.Flags().StringVarP(&url, "url", "u", "", "url to override module detection (optional)")

	return clipboardCmd
}
