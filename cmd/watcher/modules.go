package watcher

import (
	"fmt"

	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/spf13/cobra"
)

// addModulesCommand adds the modules sub command
func (cli *CliApplication) addModulesCommand() {
	modulesCmd := &cobra.Command{
		Use:   "module [module name]",
		Short: "lists the module specific commands and settings",
		Long:  "lists the module specific commands and settings which are not shared across all modules.",
	}

	moduleFactory := modules.GetModuleFactory()
	for _, module := range moduleFactory.GetAllModules() {
		moduleCmd := &cobra.Command{
			Use:   module.Key,
			Short: fmt.Sprintf("specific commands and settings of module: %s", module.Key),
		}
		module.AddModuleCommand(moduleCmd)
		modulesCmd.AddCommand(moduleCmd)
	}

	cli.rootCmd.AddCommand(modulesCmd)
}
