package watcher

import (
	"github.com/spf13/cobra"
)

// addGenerateAutoCompletionCommand adds the generate-autocomplete sub command
func (cli *CliApplication) addBackupCommand() {
	var backupCmd = &cobra.Command{
		Use:   "backup [archive name]",
		Short: "generates a backup of the current settings and database file",
		Long: "generates a zip/tar.gz file of the current settings and database file.\n" +
			"It is possible to narrow it down to specific elements like accounts/items/settings.",
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cli.config.Backup.Database.Accounts.Enabled = true
			cli.config.Backup.Database.Items.Enabled = true
			cli.config.Backup.Settings = true
			cli.watcher.Backup(args[0], cli.config)
		},
	}
	cli.addBackupArchiveFlags(backupCmd)
	backupCmd.Flags().BoolVar(&cli.config.Backup.Database.SQL, "sql", false, "generate a .sql file")

	backupCmd.AddCommand(cli.getBackupAccountsCommand())
	backupCmd.AddCommand(cli.getBackupItemsCommand())
	backupCmd.AddCommand(cli.getBackupSettingsCommand())
	cli.rootCmd.AddCommand(backupCmd)
}

// addBackupArchiveFlags adds the archive options to the local flags to reuse them on sub commands
func (cli *CliApplication) addBackupArchiveFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&cli.config.Backup.Archive.Zip, "zip", false, "use a zip(.zip) archive")
	cmd.Flags().BoolVar(&cli.config.Backup.Archive.Tar, "tar", false, "use a tar(.tar) archive")
	cmd.Flags().BoolVar(&cli.config.Backup.Archive.Gzip, "gzip", false, "use a gzip(.tar.gz) archive")
}

// getBackupAccountsCommand returns the command for the backup accounts sub command
func (cli *CliApplication) getBackupAccountsCommand() *cobra.Command {
	backupAccountsCmd := &cobra.Command{
		Use:   "accounts [archive name]",
		Short: "generates a backup of the current accounts",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cli.config.Backup.Database.Accounts.Enabled = true
			cli.watcher.Backup(args[0], cli.config)
		},
	}
	cli.addBackupArchiveFlags(backupAccountsCmd)
	return backupAccountsCmd
}

// getBackupItemsCommand returns the command for the backup items sub command
func (cli *CliApplication) getBackupItemsCommand() *cobra.Command {
	backupItemsCmd := &cobra.Command{
		Use:   "items [archive name]",
		Short: "generates a backup of the current items",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cli.config.Backup.Database.Items.Enabled = true
			cli.watcher.Backup(args[0], cli.config)
		},
	}
	cli.addBackupArchiveFlags(backupItemsCmd)
	return backupItemsCmd
}

// getBackupSettingsCommand returns the command for the backup settings sub command
func (cli *CliApplication) getBackupSettingsCommand() *cobra.Command {
	backupSettingsCmd := &cobra.Command{
		Use:   "settings [archive name]",
		Short: "generates a backup of the current settings",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cli.config.Backup.Settings = true
			cli.watcher.Backup(args[0], cli.config)
		},
	}
	return backupSettingsCmd
}
