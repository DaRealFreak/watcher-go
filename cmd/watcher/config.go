package watcher

import (
	"fmt"
	"sort"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/DaRealFreak/watcher-go/internal/settings"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// addConfigCommand registers the unified `config` command for managing all
// persisted settings (module + global blocks) from one place.
func (cli *CliApplication) addConfigCommand() {
	root := &cobra.Command{
		Use:   "config",
		Short: "view and change all watcher settings from one place",
		Long: "view and change your watcher settings (module settings, crawljob,\n" +
			"download directory, proxy limits, ...) from a single command.\n" +
			"run 'watcher config list' to discover the exact key for any setting.",
	}

	root.AddCommand(configListCommand())
	root.AddCommand(configGetCommand())
	root.AddCommand(configSetCommand())
	root.AddCommand(configListAddCommand())
	root.AddCommand(configListRemoveCommand())
	root.AddCommand(proxyLimitsCommand())

	cli.rootCmd.AddCommand(root)
}

func configListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list [filter]",
		Short: "list all settings with their current values, grouped by source",
		Args:  cobra.MaximumNArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			filter := ""
			if len(args) == 1 {
				filter = strings.ToLower(args[0])
			}
			reg := settings.Build()

			// group entries by Group, preserving a stable group order.
			groups := map[string][]settings.Entry{}
			var order []string
			for _, e := range reg.Entries() {
				if filter != "" && !strings.Contains(strings.ToLower(e.Key), filter) &&
					!strings.Contains(strings.ToLower(e.Group), filter) {
					continue
				}
				if _, seen := groups[e.Group]; !seen {
					order = append(order, e.Group)
				}
				groups[e.Group] = append(groups[e.Group], e)
			}
			// keep "global" and "crawljob" first, modules alphabetically after.
			sort.SliceStable(order, func(i, j int) bool {
				return groupRank(order[i]) < groupRank(order[j])
			})

			for _, g := range order {
				switch g {
				case "global":
					fmt.Println("[global]")
				case "crawljob":
					fmt.Println("[crawljob]")
				default:
					fmt.Printf("[module: %s]\n", g)
				}
				for _, e := range groups[g] {
					if e.ReadOnly {
						fmt.Printf("  %-55s %-10s (complex — edit via \"watcher module %s proxies ...\" or the config file)\n",
							e.Key, settings.FriendlyType(e.Type), e.Group)
						continue
					}
					fmt.Printf("  %-55s %-10s %v\n", e.Key, settings.FriendlyType(e.Type), reg.EffectiveValue(e))
				}
				if g == "global" && (filter == "" || strings.Contains("run.proxy_connection_limits", filter)) {
					fmt.Printf("  %-55s %-10s (edit via \"config proxy-limits\")\n",
						"run.proxy_connection_limits", "[]struct")
				}
			}
		},
	}
}

func groupRank(g string) int {
	switch g {
	case "global":
		return 0
	case "crawljob":
		return 1
	default:
		return 2
	}
}

func configGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get [key]",
		Short: "print one setting's effective value",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			reg := settings.Build()
			e, ok := reg.Resolve(args[0])
			if !ok {
				unknownKey(args[0])
				return
			}
			fmt.Printf("%s = %v\n", e.Key, reg.EffectiveValue(*e))
		},
	}
}

func configSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set [key] [value]",
		Short: "set a setting's value",
		Args:  cobra.ExactArgs(2),
		Run: func(_ *cobra.Command, args []string) {
			key, value := args[0], args[1]
			reg := settings.Build()
			e, ok := reg.Resolve(key)
			if !ok {
				unknownKey(key)
				return
			}
			if e.ReadOnly {
				fmt.Printf("%q is a complex setting; edit it via \"watcher module %s proxies ...\" or the config file\n", e.Key, e.Group)
				return
			}
			parsed, err := settings.ParseValue(value, e.Type)
			if err != nil {
				fmt.Printf("invalid value for %s (expected %s): %s\n", e.Key, settings.FriendlyType(e.Type), err)
				return
			}
			viper.Set(e.Key, parsed)
			raven.CheckError(viper.WriteConfig())
			if e.Kind == settings.KindStringList {
				fmt.Printf("set %s = %v (use \"config list-add/list-remove\" to edit entries individually)\n", e.Key, parsed)
				return
			}
			fmt.Printf("set %s = %s\n", e.Key, value)
		},
	}
}

func configListAddCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list-add [key] [value]",
		Short: "append a value to a list ([]string) setting",
		Args:  cobra.ExactArgs(2),
		Run:   func(_ *cobra.Command, args []string) { mutateList(args[0], args[1], true) },
	}
}

func configListRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "list-remove [key] [value]",
		Aliases: []string{"list-rm"},
		Short:   "remove a value from a list ([]string) setting",
		Args:    cobra.ExactArgs(2),
		Run:     func(_ *cobra.Command, args []string) { mutateList(args[0], args[1], false) },
	}
}

func mutateList(key, value string, add bool) {
	reg := settings.Build()
	e, ok := reg.Resolve(key)
	if !ok {
		unknownKey(key)
		return
	}
	if e.Kind != settings.KindStringList {
		fmt.Printf("%q is a %s, not a list; use \"config set\"\n", e.Key, settings.FriendlyType(e.Type))
		return
	}
	cur := viper.GetStringSlice(e.Key)
	if add {
		out, added := settings.AddToList(cur, value)
		if !added {
			fmt.Printf("%q already in %s\n", strings.TrimSpace(value), e.Key)
			return
		}
		viper.Set(e.Key, out)
		raven.CheckError(viper.WriteConfig())
		fmt.Printf("added %q to %s\n", strings.TrimSpace(value), e.Key)
		return
	}
	out, removed := settings.RemoveFromList(cur, value)
	if !removed {
		fmt.Printf("%q not in %s\n", strings.TrimSpace(value), e.Key)
		return
	}
	viper.Set(e.Key, out)
	raven.CheckError(viper.WriteConfig())
	fmt.Printf("removed %q from %s\n", strings.TrimSpace(value), e.Key)
}

func unknownKey(key string) {
	fmt.Printf("unknown setting %q; run \"watcher config list\" to see available settings\n", key)
}
