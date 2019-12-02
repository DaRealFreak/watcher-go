package models

import (
	"fmt"
	"github.com/DaRealFreak/watcher-go/pkg/http"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// AddProxyCommands adds the module specific commands for the proxy server
func (t *Module) AddProxyCommands(command *cobra.Command) {
	var proxySettings http.ProxySettings

	proxyCmd := &cobra.Command{
		Use:   "proxy",
		Short: "proxy configurations",
		Long:  "options to configure proxy settings used for the module",
		Run: func(cmd *cobra.Command, args []string) {
			// enable proxy after changing the settings
			proxySettings.Enable = true

			viper.Set(fmt.Sprintf("Modules.%s.Proxy", t.GetViperModuleKey()), proxySettings)
			raven.CheckError(viper.WriteConfig())
		},
	}

	proxyCmd.Flags().StringVarP(
		&proxySettings.Host,
		"host", "H", "",
		"host of the proxy server (required)",
	)
	proxyCmd.Flags().IntVarP(
		&proxySettings.Port,
		"port", "P", 1080,
		"port of the proxy server",
	)
	proxyCmd.Flags().StringVarP(
		&proxySettings.Username,
		"user", "u", "",
		"username for the proxy server",
	)
	proxyCmd.Flags().StringVarP(
		&proxySettings.Password,
		"password", "p", "",
		"password for the proxy server",
	)

	_ = proxyCmd.MarkFlagRequired("host")

	// add sub commands for proxy command
	t.addEnableProxyCommand(proxyCmd)
	t.addDisableProxyCommand(proxyCmd)

	// add proxy command to parent command
	command.AddCommand(proxyCmd)
}

// addEnableProxyCommand adds the proxy enable sub command
func (t *Module) addEnableProxyCommand(command *cobra.Command) {
	enableCmd := &cobra.Command{
		Use:   "enable",
		Short: "enables proxy usage",
		Long:  "option to enable proxy server usage again, after it got manually disabled",
		Run: func(cmd *cobra.Command, args []string) {
			viper.Set(fmt.Sprintf("Modules.%s.Proxy.Enable", t.GetViperModuleKey()), true)
			raven.CheckError(viper.WriteConfig())
		},
	}
	command.AddCommand(enableCmd)
}

// addDisableProxyCommand adds the proxy disable sub command
func (t *Module) addDisableProxyCommand(command *cobra.Command) {
	enableCmd := &cobra.Command{
		Use:   "disable",
		Short: "disables proxy usage",
		Long:  "option to disable proxy server usage",
		Run: func(cmd *cobra.Command, args []string) {
			viper.Set(fmt.Sprintf("Modules.%s.Proxy.Enable", t.GetViperModuleKey()), false)
			raven.CheckError(viper.WriteConfig())
		},
	}
	command.AddCommand(enableCmd)
}

// GetProxySettings returns the proxy settings for the module
func (t *Module) GetProxySettings() (proxySettings *http.ProxySettings) {
	err := viper.UnmarshalKey(
		fmt.Sprintf("Modules.%s.Proxy", t.GetViperModuleKey()),
		&proxySettings,
	)

	raven.CheckError(err)

	return proxySettings
}

