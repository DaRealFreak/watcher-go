package watcher

import (
	"github.com/DaRealFreak/watcher-go/pkg/modules"
	"github.com/spf13/cobra"
)

// addModulesCommand adds the modules sub command
func (cli *CliApplication) addModulesCommand() {
	var moduleCmd = &cobra.Command{
		Use:   "module [module name]",
		Short: "lists the module specific commands and settings",
		Long:  "lists the module specific commands and settings which are not shared across all modules.",
	}

	moduleFactory := modules.NewModuleFactory(nil)
	for _, module := range moduleFactory.GetAllModules() {
		module.AddSettingsCommand(moduleCmd)
	}

	cli.rootCmd.AddCommand(moduleCmd)
}
