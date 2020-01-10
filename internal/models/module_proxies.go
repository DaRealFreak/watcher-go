package models

import (
	"fmt"

	"github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ProxyLoopConfiguration contains the configuration for loop proxy configurations
type ProxyLoopConfiguration struct {
	Loop        bool                 `mapstructure:"loop"`
	Proxy       http.ProxySettings   `mapstructure:"proxy"`
	LoopProxies []http.ProxySettings `mapstructure:"loopproxies"`
}

// AddProxyLoopCommands adds the proxies sub command for the passed command
func (t *Module) AddProxyLoopCommands(command *cobra.Command) {
	proxiesCmd := &cobra.Command{
		Use:   "proxies",
		Short: "proxy loop configuration",
		Long:  "options to configure proxy loops",
	}

	t.addProxiesLoopCommand(proxiesCmd)
	t.addProxiesAddCommand(proxiesCmd)
	t.addProxiesRemoveCommand(proxiesCmd)
	t.addProxiesEnableCommand(proxiesCmd)
	t.addProxiesDisableCommand(proxiesCmd)
	command.AddCommand(proxiesCmd)
}

func (t *Module) addProxiesEnableCommand(command *cobra.Command) {
	var (
		proxySettings http.ProxySettings
		moduleCfg     ProxyLoopConfiguration
	)

	enableCmd := &cobra.Command{
		Use:   "enable",
		Short: "enable specific loop proxy",
		Long:  "options to enable specified loop proxy",
		Run: func(cmd *cobra.Command, args []string) {
			for _, proxy := range moduleCfg.LoopProxies {
				if proxy.Host == proxySettings.Host && proxy.Port == proxySettings.Port {
					proxy.Enable = true
					break
				}
			}

			viper.Set(
				fmt.Sprintf("Modules.%s", t.GetViperModuleKey()),
				&moduleCfg,
			)
		},
	}

	enableCmd.Flags().StringVarP(
		&proxySettings.Host,
		"host", "H", "",
		"host of the proxy server (required)",
	)
	enableCmd.Flags().IntVarP(
		&proxySettings.Port,
		"port", "P", 1080,
		"port of the proxy server",
	)

	_ = enableCmd.MarkFlagRequired("host")

	command.AddCommand(enableCmd)
}

func (t *Module) addProxiesDisableCommand(command *cobra.Command) {
	var (
		proxySettings http.ProxySettings
		moduleCfg     ProxyLoopConfiguration
	)

	disableCmd := &cobra.Command{
		Use:   "disable",
		Short: "disable specific loop proxy",
		Long:  "options to disable specified loop proxy",
		Run: func(cmd *cobra.Command, args []string) {
			for _, proxy := range moduleCfg.LoopProxies {
				if proxy.Host == proxySettings.Host && proxy.Port == proxySettings.Port {
					proxy.Enable = false
					break
				}
			}

			viper.Set(
				fmt.Sprintf("Modules.%s", t.GetViperModuleKey()),
				&moduleCfg,
			)
		},
	}

	disableCmd.Flags().StringVarP(
		&proxySettings.Host,
		"host", "H", "",
		"host of the proxy server (required)",
	)
	disableCmd.Flags().IntVarP(
		&proxySettings.Port,
		"port", "P", 1080,
		"port of the proxy server",
	)

	_ = disableCmd.MarkFlagRequired("host")

	command.AddCommand(disableCmd)
}

func (t *Module) addProxiesLoopCommand(command *cobra.Command) {
	loopCmd := &cobra.Command{
		Use:   "loop",
		Short: "configure loop",
		Long:  "options to enable/disable proxy loops",
	}

	enableCmd := &cobra.Command{
		Use:   "enable",
		Short: "enables the proxy loop",
		Run: func(cmd *cobra.Command, args []string) {
			// enable proxy after changing the settings
			viper.Set(fmt.Sprintf("Modules.%s.Loop", t.GetViperModuleKey()), true)
			raven.CheckError(viper.WriteConfig())
		},
	}

	disableCmd := &cobra.Command{
		Use:   "disable",
		Short: "disables the proxy loop",
		Run: func(cmd *cobra.Command, args []string) {
			// enable proxy after changing the settings
			viper.Set(fmt.Sprintf("Modules.%s.Loop", t.GetViperModuleKey()), false)
			raven.CheckError(viper.WriteConfig())
		},
	}

	loopCmd.AddCommand(enableCmd, disableCmd)
	command.AddCommand(loopCmd)
}

func (t *Module) addProxiesAddCommand(command *cobra.Command) {
	var proxySettings http.ProxySettings

	proxyCmd := &cobra.Command{
		Use:   "add",
		Short: "adds proxy to looped proxies",
		Long:  "options to add proxy server used in proxy loop",
		Run: func(cmd *cobra.Command, args []string) {
			t.addLoopProxy(proxySettings)
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

	command.AddCommand(proxyCmd)
}

func (t *Module) addProxiesRemoveCommand(command *cobra.Command) {
	var (
		proxySettings http.ProxySettings
		moduleCfg     ProxyLoopConfiguration
	)

	proxyCmd := &cobra.Command{
		Use:   "remove",
		Short: "removes a proxy from the looped proxies",
		Long:  "options to remove a proxy server used in the proxy loop",
		Run: func(cmd *cobra.Command, args []string) {
			err := viper.UnmarshalKey(
				fmt.Sprintf("Modules.%s", t.GetViperModuleKey()),
				&moduleCfg,
			)
			raven.CheckError(err)

			for s, proxy := range moduleCfg.LoopProxies {
				if proxy.Host == proxySettings.Host && proxy.Port == proxySettings.Port {
					moduleCfg.LoopProxies = append(moduleCfg.LoopProxies[:s], moduleCfg.LoopProxies[s+1:]...)
				}
			}

			viper.Set(
				fmt.Sprintf("Modules.%s", t.GetViperModuleKey()),
				&moduleCfg,
			)
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

	_ = proxyCmd.MarkFlagRequired("host")

	command.AddCommand(proxyCmd)
}

// addLoopProxy adds the loop proxy to the list or updates the proxy settings if already added
func (t *Module) addLoopProxy(proxySettings http.ProxySettings) {
	var moduleCfg ProxyLoopConfiguration

	// enable proxy on adding action
	proxySettings.Enable = true
	err := viper.UnmarshalKey(
		fmt.Sprintf("Modules.%s", t.GetViperModuleKey()),
		&moduleCfg,
	)
	raven.CheckError(err)

	updated := false

	for _, proxy := range moduleCfg.LoopProxies {
		if proxy.Host == proxySettings.Host && proxy.Port == proxySettings.Port {
			proxy = proxySettings
			updated = true

			break
		}
	}

	if !updated {
		// add new proxy
		moduleCfg.LoopProxies = append(moduleCfg.LoopProxies, proxySettings)
	}

	viper.Set(
		fmt.Sprintf("Modules.%s", t.GetViperModuleKey()),
		&moduleCfg,
	)
	raven.CheckError(viper.WriteConfig())
}
