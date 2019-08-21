package watcher

import (
	"fmt"
	"github.com/spf13/cobra"
)

// addGenerateAutoCompletionCommand adds the generate-autocomplete sub command
func (cli *CliApplication) addBackupCommand() {
	var backupCmd = &cobra.Command{
		Use:   "backup",
		Short: "generates a backup of the current settings and database file",
		Long: "generates a zip/tar.gz file of the current settings and database file.\n" +
			"It is possible to narrow it down to specific elements like accounts/items/settings.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(cli.configuration.backup)
		},
	}
	// the archive flags are persistent for all sub commands
	backupCmd.PersistentFlags().BoolVar(&cli.configuration.backup.zip, "zip", false, "use a zip(.zip) archive")
	backupCmd.PersistentFlags().BoolVar(&cli.configuration.backup.tar, "tar", false, "use a tar(.tar) archive")
	backupCmd.PersistentFlags().BoolVar(&cli.configuration.backup.gzip, "gzip", false, "use a gzip(.tar.gz) archive")
	// use this library to dump all https://github.com/schollz/sqlite3dump
	backupCmd.PersistentFlags().BoolVar(&cli.configuration.backup.sql, "sql", false, "generate a .sql file")

	backupCmd.AddCommand(cli.getBackupAccountsCommand())
	backupCmd.AddCommand(cli.getBackupItemsCommand())
	backupCmd.AddCommand(cli.getBackupSettingsCommand())
	cli.rootCmd.AddCommand(backupCmd)
}

// getBackupAccountsCommand returns the command for the backup accounts sub command
func (cli *CliApplication) getBackupAccountsCommand() *cobra.Command {
	var url string
	backupAccountsCmd := &cobra.Command{
		Use:   "accounts",
		Short: "generates a backup of the current accounts",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("ToDo")
		},
	}
	backupAccountsCmd.Flags().StringVar(&url, "url", "", "url of module")
	return backupAccountsCmd
}

// getBackupItemsCommand returns the command for the backup items sub command
func (cli *CliApplication) getBackupItemsCommand() *cobra.Command {
	var url string
	backupItemsCmd := &cobra.Command{
		Use:   "items",
		Short: "generates a backup of the current items",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("ToDo")
		},
	}
	backupItemsCmd.Flags().StringVar(&url, "url", "", "url of module")
	return backupItemsCmd
}

// getBackupSettingsCommand returns the command for the backup settings sub command
func (cli *CliApplication) getBackupSettingsCommand() *cobra.Command {
	backupSettingsCmd := &cobra.Command{
		Use:   "settings",
		Short: "generates a backup of the current settings",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("ToDo")
		},
	}
	return backupSettingsCmd
}
