package watcher

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	watcherHttp "github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/spf13/viper"
)

// acquireModuleLeases reserves connection-budget capacity for a module's
// run. One lease is acquired per unique (proxy username, eTLD+1 domain)
// pair the module is configured to use; slot count for each is the number
// of declared proxies in that account, capped to half the pool's Max so
// any two modules sharing an account can always coexist.
//
// Modules without any configured proxy get no leases (their requests bypass
// the budget entirely, which is correct — they aren't hitting the VPN).
// Pools without a configured per-domain policy get no lease (unlimited
// domain — no need to reserve).
//
// Returned leases must be Release()d when the module's run finishes; the
// caller's defer should iterate the slice.
func acquireModuleLeases(moduleKey string) []*watcherHttp.Lease {
	if watcherHttp.GlobalLeases == nil || watcherHttp.Global == nil {
		return nil
	}

	accounts := discoverModuleAccounts(moduleKey)
	if len(accounts) == 0 {
		return nil
	}

	leases := make([]*watcherHttp.Lease, 0, len(accounts))
	for _, acc := range accounts {
		policy, ok := watcherHttp.Global.PolicyFor(acc.domain)
		if !ok || policy.Max <= 0 {
			// Domain isn't gated — no lease needed.
			continue
		}
		slots := sizeLeaseSlots(acc.proxies, policy.Max)
		lease, err := watcherHttp.GlobalLeases.AcquireLease(
			context.Background(),
			moduleKey,
			acc.key,
			slots,
			policy,
		)
		if err != nil {
			slog.Warn(
				fmt.Sprintf("failed to acquire proxy lease: %s", err.Error()),
				"module", moduleKey, "account", acc.key,
			)
			continue
		}
		slog.Debug(
			fmt.Sprintf("module acquired lease account=%s slots=%d", acc.key, lease.Slots()),
			"module", moduleKey,
		)
		leases = append(leases, lease)
	}
	return leases
}

// sizeLeaseSlots picks the slot count for a lease. Default = number of
// declared proxies, capped so a single module can't take more than half the
// pool — leaves headroom for at least one peer module to coexist.
func sizeLeaseSlots(declaredProxies, poolMax int) int {
	if declaredProxies < 1 {
		declaredProxies = 1
	}
	ceiling := poolMax / 2
	if ceiling < 1 {
		ceiling = 1
	}
	if declaredProxies > ceiling {
		return ceiling
	}
	return declaredProxies
}

// moduleAccount summarizes one (username, domain) pair a module declares
// proxies for, with the count of distinct proxy entries in that pair.
type moduleAccount struct {
	key     string // username@domain
	domain  string
	proxies int
}

// discoverModuleAccounts inspects the module's Viper config to find every
// (username, domain) pair the module's proxies use, with a count per pair.
// Reads both LoopProxies (multiproxy) and Proxy (single-proxy) settings.
func discoverModuleAccounts(moduleKey string) []moduleAccount {
	viperKey := strings.ReplaceAll(moduleKey, ".", "_")
	base := fmt.Sprintf("Modules.%s", viperKey)

	counts := make(map[string]*moduleAccount)
	tally := func(p watcherHttp.ProxySettings) {
		if !p.Enable || p.Host == "" {
			return
		}
		domain := watcherHttp.DomainFor(p.Host)
		key := p.Username + "@" + domain
		if existing, ok := counts[key]; ok {
			existing.proxies++
			return
		}
		counts[key] = &moduleAccount{key: key, domain: domain, proxies: 1}
	}

	var loopProxies []watcherHttp.ProxySettings
	_ = viper.UnmarshalKey(base+".loopproxies", &loopProxies)
	for _, p := range loopProxies {
		tally(p)
	}

	// Single-proxy entry (used by non-multiproxy or as the default).
	var single watcherHttp.ProxySettings
	_ = viper.UnmarshalKey(base+".proxy", &single)
	tally(single)

	out := make([]moduleAccount, 0, len(counts))
	for _, info := range counts {
		out = append(out, *info)
	}
	// Sort by key so every module acquires its leases in the same global
	// order. acquireModuleLeases takes leases one account at a time with an
	// uncancelable context; without a consistent ordering, two parallel
	// modules sharing two accounts could grab them in opposite order and
	// deadlock permanently. A total order over accountKey prevents the cycle.
	sort.Slice(out, func(i, j int) bool { return out[i].key < out[j].key })
	return out
}
