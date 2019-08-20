package watcher

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func (cli *CliApplication) addGenerateAutoCompletionCommand() {
	cli.rootCmd.AddCommand(cli.getGenerateAutoCompletionCommand())
}

// retrieve the generate autocomplete command
func (cli *CliApplication) getGenerateAutoCompletionCommand() *cobra.Command {
	var generateAutocompleteCmd = &cobra.Command{
		Use:   "generate-autocomplete",
		Short: "generates auto completion for Bash, Zsh and PowerShell",
	}
	generateAutocompleteCmd.AddCommand(cli.getGenerateAutoCompletionCommandBash())
	generateAutocompleteCmd.AddCommand(cli.getGenerateAutoCompletionCommandZsh())
	generateAutocompleteCmd.AddCommand(cli.getGenerateAutoCompletionCommandPowerShell())
	return generateAutocompleteCmd
}

func (cli *CliApplication) getGenerateAutoCompletionCommandPowerShell() *cobra.Command {
	var powerShellCmd = &cobra.Command{
		Use:   "powershell",
		Short: "generates auto completion for PowerShell",
		Run: func(cmd *cobra.Command, args []string) {
			buffer := new(bytes.Buffer)
			if err := cli.rootCmd.GenPowerShellCompletion(buffer); err != nil {
				log.Fatal(err)
			}

			if err := cli.createAutoCompletionFile(
				"autocomplete_powershell.ps1",
				buffer.Bytes(),
				"echo 'Import-Module -Name \"%s\"' >> $PROFILE",
			); err != nil {
				log.Fatal(err)
			}
		},
	}
	return powerShellCmd
}

func (cli *CliApplication) getGenerateAutoCompletionCommandBash() *cobra.Command {
	var bashCmd = &cobra.Command{
		Use:   "bash",
		Short: "generates auto completion for Bash",
		Run: func(cmd *cobra.Command, args []string) {
			buffer := new(bytes.Buffer)
			if err := cli.rootCmd.GenBashCompletion(buffer); err != nil {
				log.Fatal(err)
			}

			// update path to current executable to work with .exe instead of just without extension
			fContent := cli.updateBashCompletionCommandToCurrentExecutable(buffer.Bytes())

			if err := cli.createAutoCompletionFile(
				"autocomplete_bash.sh",
				fContent,
				"printf '\\nsource \"%s\"' >> ~/.bashrc",
			); err != nil {
				log.Fatal(err)
			}
		},
	}
	return bashCmd
}

func (cli *CliApplication) getGenerateAutoCompletionCommandZsh() *cobra.Command {
	var zshCmd = &cobra.Command{
		Use:   "zsh",
		Short: "generates auto completion for Zsh",
		Run: func(cmd *cobra.Command, args []string) {
			buffer := new(bytes.Buffer)
			if err := cli.rootCmd.GenZshCompletion(buffer); err != nil {
				log.Fatal(err)
			}

			if err := cli.createAutoCompletionFile(
				"autocomplete_zsh.sh",
				buffer.Bytes(),
				"printf '\\nsh \"%s\"' >> ~/.zshrc",
			); err != nil {
				log.Fatal(err)
			}
		},
	}
	return zshCmd
}

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
