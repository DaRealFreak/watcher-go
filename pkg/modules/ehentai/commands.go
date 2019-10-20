package ehentai

import (
	"github.com/DaRealFreak/watcher-go/pkg/http/session"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// addProxyCommands adds the module specific commands for the proxy server
func (m *ehentai) addProxyCommands(command *cobra.Command) {
	var proxySettings session.ProxySettings

	proxyCmd := &cobra.Command{
		Use:   "proxy",
		Short: "proxy configurations",
		Long:  "options to configure proxy settings used for the module",
		Run: func(cmd *cobra.Command, args []string) {
			// enable proxy after changing the settings
			viper.Set("Modules.ehentai.Proxy.Enable", true)
			viper.Set("Modules.ehentai.Proxy.Host", proxySettings.Address)
			viper.Set("Modules.ehentai.Proxy.Port", proxySettings.Port)
			viper.Set("Modules.ehentai.Proxy.Username", proxySettings.Username)
			viper.Set("Modules.ehentai.Proxy.Password", proxySettings.Password)
			raven.CheckError(viper.WriteConfig())
		},
	}

	proxyCmd.Flags().StringVarP(
		&proxySettings.Address,
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
	m.addEnableProxyCommand(proxyCmd)
	m.addDisableProxyCommand(proxyCmd)

	// add proxy command to parent command
	command.AddCommand(proxyCmd)
}

// addEnableProxyCommand adds the proxy enable sub command
func (m *ehentai) addEnableProxyCommand(command *cobra.Command) {
	enableCmd := &cobra.Command{
		Use:   "enable",
		Short: "enables proxy usage",
		Long:  "option to enable proxy server usage again, after it got manually disabled",
		Run: func(cmd *cobra.Command, args []string) {
			viper.Set("Modules.ehentai.Proxy.Enable", true)
			raven.CheckError(viper.WriteConfig())
		},
	}
	command.AddCommand(enableCmd)
}

// addDisableProxyCommand adds the proxy disable sub command
func (m *ehentai) addDisableProxyCommand(command *cobra.Command) {
	enableCmd := &cobra.Command{
		Use:   "disable",
		Short: "disables proxy usage",
		Long:  "option to disable proxy server usage",
		Run: func(cmd *cobra.Command, args []string) {
			viper.Set("Modules.ehentai.Proxy.Enable", false)
			raven.CheckError(viper.WriteConfig())
		},
	}
	command.AddCommand(enableCmd)
}

// getProxySettings returns the proxy settings for the module
func (m *ehentai) getProxySettings() *session.ProxySettings {
	return &session.ProxySettings{
		Use:      viper.GetBool("Modules.ehentai.Proxy.Enable"),
		Address:  viper.GetString("Modules.ehentai.Proxy.Host"),
		Port:     viper.GetInt("Modules.ehentai.Proxy.Port"),
		Username: viper.GetString("Modules.ehentai.Proxy.Username"),
		Password: viper.GetString("Modules.ehentai.Proxy.Password"),
	}
}
