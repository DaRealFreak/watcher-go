package watcher

import (
	"github.com/spf13/cobra"
)

// addRestoreCommand adds the restore sub command
func (cli *CliApplication) addRestoreCommand() {
	restoreCmd := &cobra.Command{
		Use:   "restore [archive name]",
		Short: "restores the current settings/database from the passed backup archive",
		Long: "uses the passed archive file to restore the backed up setting/database file.\n" +
			"It is possible to narrow it down to specific elements like accounts/items/settings.",
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cli.config.Restore.Database.Accounts.Enabled = true
			cli.config.Restore.Database.Items.Enabled = true
			cli.config.Restore.Settings = true
			cli.watcher.Restore(args[0], cli.config)
		},
	}

	restoreCmd.AddCommand(cli.getRestoreAccountsCommand())
	restoreCmd.AddCommand(cli.getRestoreItemsCommand())
	restoreCmd.AddCommand(cli.getRestoreSettingsCommand())
	cli.rootCmd.AddCommand(restoreCmd)
}

// getRestoreAccountsCommand returns the command for the restore accounts sub command
func (cli *CliApplication) getRestoreAccountsCommand() *cobra.Command {
	restoreAccountsCmd := &cobra.Command{
		Use:   "accounts [archive name]",
		Short: "restores the accounts table from the passed archive",
		Long: "restores only the accounts table from the passed archive.\n" +
			"Requires an account.sql file in the backup archive",
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cli.config.Restore.Database.Accounts.Enabled = true
			cli.watcher.Restore(args[0], cli.config)
		},
	}

	return restoreAccountsCmd
}

// getRestoreItemsCommand returns the command for the restore items sub command
func (cli *CliApplication) getRestoreItemsCommand() *cobra.Command {
	backupItemsCmd := &cobra.Command{
		Use:   "items [archive name]",
		Short: "restores the tracked_items table from the passed archive",
		Long: "restores only the tracked_items table from the passed archive.\n" +
			"Requires a tracked_items.sql file in the backup archive",
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cli.config.Restore.Database.Items.Enabled = true
			cli.watcher.Restore(args[0], cli.config)
		},
	}

	return backupItemsCmd
}

// getRestoreSettingsCommand returns the command for the restore settings sub command
func (cli *CliApplication) getRestoreSettingsCommand() *cobra.Command {
	backupSettingsCmd := &cobra.Command{
		Use:   "settings [archive name]",
		Short: "restores the settings file from the passed archive",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cli.config.Restore.Settings = true
			cli.watcher.Restore(args[0], cli.config)
		},
	}

	return backupSettingsCmd
}
