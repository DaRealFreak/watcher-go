package watcher

import (
	"testing"

	"github.com/spf13/viper"
)

// TestDiscoverModuleAccounts_DeterministicOrder is a regression test for the
// lease-acquisition deadlock: acquireModuleLeases takes leases one account at a
// time with an uncancelable context, so every module must request its leases in
// the same global order. discoverModuleAccounts must therefore return its
// accounts sorted by key regardless of Viper/map iteration order.
func TestDiscoverModuleAccounts_DeterministicOrder(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	viper.Set("Modules.testmodule.loopproxies", []map[string]any{
		{"enable": true, "host": "us1.proxy.nordvpn.com", "username": "bob"},
		{"enable": true, "host": "us1.mullvad.net", "username": "alice"},
		// second nordvpn entry collapses into the same account as the first
		{"enable": true, "host": "us2.proxy.nordvpn.com", "username": "bob"},
	})

	accounts := discoverModuleAccounts("testmodule")
	if len(accounts) != 2 {
		t.Fatalf("expected 2 distinct accounts, got %d: %+v", len(accounts), accounts)
	}

	for i := 1; i < len(accounts); i++ {
		if accounts[i-1].key > accounts[i].key {
			t.Fatalf("accounts not sorted by key: %q before %q", accounts[i-1].key, accounts[i].key)
		}
	}

	if accounts[0].key != "alice@mullvad.net" {
		t.Fatalf("expected first account alice@mullvad.net, got %q", accounts[0].key)
	}
	if accounts[1].key != "bob@nordvpn.com" || accounts[1].proxies != 2 {
		t.Fatalf("expected bob@nordvpn.com with 2 proxies, got %q proxies=%d",
			accounts[1].key, accounts[1].proxies)
	}
}
