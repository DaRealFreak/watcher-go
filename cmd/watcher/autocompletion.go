package watcher

import (
	"bytes"
	"fmt"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
)

// addGenerateAutoCompletionCommand adds the generate-autocomplete sub command
func (cli *CliApplication) addGenerateAutoCompletionCommand() {
	var generateAutocompleteCmd = &cobra.Command{
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
	var powerShellCmd = &cobra.Command{
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
	var bashCmd = &cobra.Command{
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
	var zshCmd = &cobra.Command{
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

// createAutoCompletionFile creates the auto completion file in the default directory
// and prints the activiation command if successfully created
func (cli *CliApplication) createAutoCompletionFile(fName string, fContent []byte, activationCmd string) (err error) {
	if dir, err := homedir.Dir(); err == nil {
		if err := os.MkdirAll(filepath.Join(dir, ".watcher", "completion"), os.ModePerm); err != nil {
			return err
		}
		filePath := filepath.ToSlash(filepath.Join(dir, ".watcher", "completion", fName))
		if err := ioutil.WriteFile(filePath, fContent, os.ModePerm); err != nil {
			return err
		}
		fmt.Println("auto completion script got created at: " + filePath)
		fmt.Print(
			fmt.Sprintf(
				"run the following command to add the auto completion to your profile: \n%s",
				fmt.Sprintf(activationCmd, filePath),
			),
		)
	}
	return err
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
