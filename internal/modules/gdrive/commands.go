package gdrive

import (
	"fmt"

	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// AddProxyCommands adds the module specific commands for the proxy server
func (m *gdrive) addAuthenticationCommands(command *cobra.Command) {
	proxyCmd := &cobra.Command{
		Use:   "auth [filepath]",
		Short: "service authentication options",
		Long:  "options to configure service authentication options used for google drive",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			viper.Set(fmt.Sprintf("Modules.%s.service_json_path", m.GetViperModuleKey()), args[0])
			raven.CheckError(viper.WriteConfig())
		},
	}

	// add proxy command to parent command
	command.AddCommand(proxyCmd)
}
