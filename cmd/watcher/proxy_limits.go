package watcher

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/DaRealFreak/watcher-go/internal/configuration"
	watcherHttp "github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// addProxyLimitsCommand registers the `proxy-limits` command group used to
// manage the per-(account, domain) proxy connection budget from the CLI.
func (cli *CliApplication) addProxyLimitsCommand() {
	root := &cobra.Command{
		Use:   "proxy-limits",
		Short: "manage per-service proxy connection limits",
		Long: "manage the global proxy connection budget that caps simultaneous\n" +
			"in-flight HTTP requests per (proxy username, host eTLD+1) pool.",
	}

	root.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "list configured proxy connection limits",
		Run: func(cmd *cobra.Command, args []string) {
			entries := readProxyLimitsList()
			if len(entries) == 0 {
				fmt.Println("no proxy connection limits configured")
				return
			}
			fmt.Printf("%-30s %-6s %s\n", "DOMAIN", "MAX", "COOLDOWN")
			for _, e := range entries {
				fmt.Printf("%-30s %-6d %ds\n", e.Domain, e.Max, e.CooldownSeconds)
			}
		},
	})

	root.AddCommand(&cobra.Command{
		Use:   "set [domain] [max] [cooldown_seconds]",
		Short: "set or update the connection cap for a domain",
		Long: "set the per-account cap for an eTLD+1 domain. an optional cooldown\n" +
			"(in seconds) delays handing a freshly-released slot to a different host\n" +
			"in the same pool — gives the VPN's server-side counter time to settle.\n" +
			"omit the cooldown to preserve the existing value (0 by default).\n" +
			"examples:\n" +
			"  watcher proxy-limits set nordvpn.com 10\n" +
			"  watcher proxy-limits set nordvpn.com 10 10",
		Args: cobra.RangeArgs(2, 3),
		Run: func(cmd *cobra.Command, args []string) {
			domain := args[0]
			max, err := strconv.Atoi(args[1])
			if err != nil || max <= 0 {
				fmt.Printf("invalid max %q: must be a positive integer\n", args[1])
				return
			}

			var cooldown *int
			if len(args) >= 3 {
				cd, err := strconv.Atoi(args[2])
				if err != nil || cd < 0 {
					fmt.Printf("invalid cooldown %q: must be a non-negative integer\n", args[2])
					return
				}
				cooldown = &cd
			}

			entries := readProxyLimitsList()
			replaced := false
			for i := range entries {
				if entries[i].Domain == domain {
					entries[i].Max = max
					if cooldown != nil {
						entries[i].CooldownSeconds = *cooldown
					}
					replaced = true
					break
				}
			}
			if !replaced {
				entry := configuration.ProxyConnectionLimit{Domain: domain, Max: max}
				if cooldown != nil {
					entry.CooldownSeconds = *cooldown
				}
				entries = append(entries, entry)
			}

			writeProxyLimitsList(entries)
			action := "added"
			if replaced {
				action = "updated"
			}
			cd := 0
			for _, e := range entries {
				if e.Domain == domain {
					cd = e.CooldownSeconds
					break
				}
			}
			fmt.Printf("%s %s: max=%d cooldown=%ds\n", action, domain, max, cd)
		},
	})

	root.AddCommand(&cobra.Command{
		Use:     "remove [domain]",
		Aliases: []string{"rm"},
		Short:   "remove the connection cap for a domain",
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			domain := args[0]
			entries := readProxyLimitsList()
			removed := false
			out := entries[:0]
			for _, e := range entries {
				if e.Domain == domain {
					removed = true
					continue
				}
				out = append(out, e)
			}
			if !removed {
				fmt.Printf("no limit configured for %s\n", domain)
				return
			}
			writeProxyLimitsList(out)
			fmt.Printf("removed %s\n", domain)
		},
	})

	cli.rootCmd.AddCommand(root)
}

// loadProxyConnectionLimits reads run.proxy_connection_limits from viper and
// returns a domain -> policy map suitable for ConnectionBudget.
//
// Expected YAML shape:
//
//	run:
//	  proxy_connection_limits:
//	    - domain: nordvpn.com
//	      max: 10
//	      cooldown_seconds: 10
//	    - domain: mullvad.net
//	      max: 5
func loadProxyConnectionLimits() map[string]watcherHttp.DomainPolicy {
	out := make(map[string]watcherHttp.DomainPolicy)
	for _, l := range readProxyLimitsList() {
		out[l.Domain] = watcherHttp.DomainPolicy{
			Max:      l.Max,
			Cooldown: time.Duration(l.CooldownSeconds) * time.Second,
		}
	}
	return out
}

// readProxyLimitsList returns the current limits from viper as a filtered,
// alphabetically-sorted slice suitable for editing and writing back.
func readProxyLimitsList() []configuration.ProxyConnectionLimit {
	var raw []configuration.ProxyConnectionLimit
	_ = viper.UnmarshalKey("run.proxy_connection_limits", &raw)

	out := raw[:0]
	for _, e := range raw {
		if e.Domain != "" && e.Max > 0 {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Domain < out[j].Domain })
	return out
}

// writeProxyLimitsList serializes the entries back to viper as a list of
// {domain, max, cooldown_seconds} maps and persists the config. Using
// []map[string]any (rather than the struct slice directly) guarantees viper
// writes the YAML in the canonical list shape rather than the struct field's
// Go casing. cooldown_seconds is omitted from the output when zero so the
// common case stays uncluttered.
func writeProxyLimitsList(entries []configuration.ProxyConnectionLimit) {
	serialized := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		m := map[string]any{
			"domain": e.Domain,
			"max":    e.Max,
		}
		if e.CooldownSeconds > 0 {
			m["cooldown_seconds"] = e.CooldownSeconds
		}
		serialized = append(serialized, m)
	}
	viper.Set("run.proxy_connection_limits", serialized)
	raven.CheckError(viper.WriteConfig())
}
