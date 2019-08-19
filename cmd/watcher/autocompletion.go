package watcher

import (
	"fmt"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
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
			if dir, err := homedir.Dir(); err == nil {
				if err := os.MkdirAll(filepath.Join(dir, ".watcher", "completion"), os.ModePerm); err != nil {
					log.Fatal(err)
				}
				filePath := filepath.ToSlash(filepath.Join(dir, ".watcher", "completion", "autocomplete_powershell.ps1"))
				if err := cli.rootCmd.GenPowerShellCompletionFile(filePath); err != nil {
					log.Fatal(err)
				}
				fmt.Println("auto completion script got created at: " + filePath)
				fmt.Println(
					fmt.Sprintf(
						"run the following command to add the auto completion to your profile: \n%s",
						"echo 'Import-Module -Name \""+filePath+"\"' >> $PROFILE",
					),
				)
			} else {
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
			if dir, err := homedir.Dir(); err == nil {
				if err := os.MkdirAll(filepath.Join(dir, ".watcher", "completion"), os.ModePerm); err != nil {
					log.Fatal(err)
				}
				filePath := filepath.ToSlash(filepath.Join(dir, ".watcher", "completion", "autocomplete_bash.sh"))
				if err := cli.rootCmd.GenBashCompletionFile(filePath); err != nil {
					log.Fatal(err)
				}
				// update path to current executable to work with .exe instead of just without extension
				cli.updateBashCompletionCommandToCurrentExecutable(filePath)
				fmt.Println("auto completion script got created at: " + filePath)
				fmt.Println(
					fmt.Sprintf(
						"run the following command to add the auto completion to your profile: \n%s",
						"printf '\\nsource \""+filePath+"\"' >> ~/.bashrc",
					),
				)
			} else {
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
			if dir, err := homedir.Dir(); err == nil {
				if err := os.MkdirAll(filepath.Join(dir, ".watcher", "completion"), os.ModePerm); err != nil {
					log.Fatal(err)
				}
				filePath := filepath.ToSlash(filepath.Join(dir, ".watcher", "completion", "autocomplete_zsh.sh"))
				if err := cli.rootCmd.GenBashCompletionFile(filePath); err != nil {
					log.Fatal(err)
				}
				fmt.Println("auto completion script got created at: " + filePath)
				fmt.Print(
					fmt.Sprintf(
						"run the following command to add the auto completion to your profile: \n%s",
						"printf '\\nsh \""+filePath+"\"' >> ~/.zshrc",
					),
				)
			} else {
				log.Fatal(err)
			}
		},
	}
	return zshCmd
}

func (cli *CliApplication) updateBashCompletionCommandToCurrentExecutable(filePath string) {
	if executable, err := os.Executable(); err == nil {
		executable = filepath.Base(executable)
		fileContent, err := ioutil.ReadFile(filePath)
		if err != nil {
			log.Fatal(err)
		}

		newContents := strings.Replace(
			string(fileContent),
			fmt.Sprintf(" -F __start_%s %s", cli.rootCmd.Use, cli.rootCmd.Use),
			fmt.Sprintf(" -F __start_%s %s", cli.rootCmd.Use, executable),
			-1,
		)

		if err := ioutil.WriteFile(filePath, []byte(newContents), 0); err != nil {
			log.Fatal(err)
		}
	}
}
