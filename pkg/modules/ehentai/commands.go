package ehentai

import (
	"fmt"
	"github.com/DaRealFreak/watcher-go/pkg/http/session"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type ProxyConfiguration struct {
	Loop        bool                    `mapstructure:"loop"`
	Proxy       session.ProxySettings   `mapstructure:"proxy"`
	LoopProxies []session.ProxySettings `mapstructure:"proxies"`
}

func (m *ehentai) addProxyLoopCommands(command *cobra.Command) {
	proxiesCmd := &cobra.Command{
		Use:   "proxies",
		Short: "proxy loop configuration",
		Long:  "options to configure proxy loops",
	}

	m.addProxyLoopCommand(proxiesCmd)
	m.addProxyLoopProxiesCommand(proxiesCmd)
	command.AddCommand(proxiesCmd)
}

func (m *ehentai) addProxyLoopCommand(command *cobra.Command) {
	var disableLoop bool

	loopCmd := &cobra.Command{
		Use:   "loop",
		Short: "configure loop",
		Long:  "options to enable/disable proxy loops",
		Run: func(cmd *cobra.Command, args []string) {
			// enable proxy after changing the settings
			viper.Set(fmt.Sprintf("Modules.%s.Proxies.Loop", m.GetViperModuleKey()), !disableLoop)
			raven.CheckError(viper.WriteConfig())
		},
	}

	loopCmd.Flags().BoolVar(
		&disableLoop,
		"disable",
		false,
		"host of the proxy server (required)",
	)

	command.AddCommand(loopCmd)
}

func (m *ehentai) addProxyLoopProxiesCommand(command *cobra.Command) {
	var (
		proxySettings   session.ProxySettings
		existingProxies ProxyConfiguration
	)

	proxyCmd := &cobra.Command{
		Use:   "add",
		Short: "adds proxy to looped proxies",
		Long:  "options to add proxy server used in proxy loop",
		Run: func(cmd *cobra.Command, args []string) {
			// enable proxy after changing the settings
			raven.CheckError(viper.WriteConfig())
			err := viper.UnmarshalKey(
				fmt.Sprintf("Modules.%s", m.GetViperModuleKey()),
				&existingProxies,
			)
			fmt.Println(err)
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

	command.AddCommand(proxyCmd)
}
