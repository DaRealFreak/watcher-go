package watcher

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// addGenerateAutoCompletionCommand adds the generate-autocomplete sub command
func (cli *CliApplication) addGenerateAutoCompletionCommand() {
	generateAutocompleteCmd := &cobra.Command{
		Use:   "generate-autocomplete",
		Short: "generates auto completion for Bash, Zsh and PowerShell",
	}

	generateAutocompleteCmd.AddCommand(cli.getGenerateAutoCompletionCommandBash())
	generateAutocompleteCmd.AddCommand(cli.getGenerateAutoCompletionCommandZsh())
	generateAutocompleteCmd.AddCommand(cli.getGenerateAutoCompletionCommandPowerShell())
	cli.rootCmd.AddCommand(generateAutocompleteCmd)
}

// getGenerateAutoCompletionCommandPowerShell returns the command for the generate-autocomplete powershell sub command
func (cli *CliApplication) getGenerateAutoCompletionCommandPowerShell() *cobra.Command {
	powerShellCmd := &cobra.Command{
		Use:   "powershell",
		Short: "generates auto completion for PowerShell",
		Run: func(cmd *cobra.Command, args []string) {
			buffer := new(bytes.Buffer)
			err := cli.rootCmd.GenPowerShellCompletion(buffer)
			raven.CheckError(err)

			err = cli.createAutoCompletionFile(
				"autocomplete_powershell.ps1",
				buffer.Bytes(),
				"echo 'Import-Module -Name \"%s\"' >> $PROFILE",
			)
			raven.CheckError(err)
		},
	}

	return powerShellCmd
}

// getGenerateAutoCompletionCommandBash returns the command for the generate-autocomplete bash sub command
func (cli *CliApplication) getGenerateAutoCompletionCommandBash() *cobra.Command {
	bashCmd := &cobra.Command{
		Use:   "bash",
		Short: "generates auto completion for Bash",
		Run: func(cmd *cobra.Command, args []string) {
			buffer := new(bytes.Buffer)
			err := cli.rootCmd.GenBashCompletion(buffer)
			raven.CheckError(err)

			fContent := cli.updateBashCompletionCommandToCurrentExecutable(buffer.Bytes())

			err = cli.createAutoCompletionFile(
				"autocomplete_bash.sh",
				fContent,
				"printf '\\nsource \"%s\"' >> ~/.bashrc",
			)
			raven.CheckError(err)
		},
	}

	return bashCmd
}

// getGenerateAutoCompletionCommandZsh returns the command for the generate-autocomplete zsh sub command
func (cli *CliApplication) getGenerateAutoCompletionCommandZsh() *cobra.Command {
	zshCmd := &cobra.Command{
		Use:   "zsh",
		Short: "generates auto completion for Zsh",
		Run: func(cmd *cobra.Command, args []string) {
			buffer := new(bytes.Buffer)
			err := cli.rootCmd.GenZshCompletion(buffer)
			raven.CheckError(err)

			err = cli.createAutoCompletionFile(
				"autocomplete_zsh.sh",
				buffer.Bytes(),
				"printf '\\nsh \"%s\"' >> ~/.zshrc",
			)
			raven.CheckError(err)
		},
	}

	return zshCmd
}

// createAutoCompletionFile creates the autocompletion file in the default directory
// and prints the activation command if successfully created
func (cli *CliApplication) createAutoCompletionFile(fName string, fContent []byte, activationCmd string) error {
	if dir, err := homedir.Dir(); err != nil {
		return err
	} else {
		if err := os.MkdirAll(filepath.Join(dir, ".watcher", "completion"), os.ModePerm); err != nil {
			return err
		}

		filePath := filepath.ToSlash(filepath.Join(dir, ".watcher", "completion", fName))
		if err := ioutil.WriteFile(filePath, fContent, os.ModePerm); err != nil {
			return err
		}

		log.Info("auto completion script got created at: " + filePath)
		log.Info(
			fmt.Sprintf(
				"run the following command to add the auto completion to your profile: \n%s",
				fmt.Sprintf(activationCmd, filePath),
			),
		)

		return nil
	}
}

// updateBashCompletionCommandToCurrentExecutable updates activation path to current executable
// to also work with .exe on Windows instead of just the default Use
func (cli *CliApplication) updateBashCompletionCommandToCurrentExecutable(fileContent []byte) []byte {
	if executable, err := os.Executable(); err == nil {
		executable = filepath.Base(executable)
		newContents := strings.Replace(
			string(fileContent),
			fmt.Sprintf(" -F __start_%s %s", cli.rootCmd.Use, cli.rootCmd.Use),
			fmt.Sprintf(" -F __start_%s %s", cli.rootCmd.Use, executable),
			-1,
		)

		return []byte(newContents)
	}

	return fileContent
}
