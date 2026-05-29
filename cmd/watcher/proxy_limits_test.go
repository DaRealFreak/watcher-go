package watcher

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DaRealFreak/watcher-go/internal/configuration"
	watcherHttp "github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/spf13/viper"
)

// TestLoadProxyConnectionLimits exercises the production loader against the
// supported YAML shape plus negative / filtering cases.
func TestLoadProxyConnectionLimits(t *testing.T) {
	cases := []struct {
		name string
		body string
		want map[string]watcherHttp.DomainPolicy
	}{
		{
			name: "list of structs (preferred)",
			body: `run:
  proxy_connection_limits:
    - domain: nordvpn.com
      max: 10
    - domain: mullvad.net
      max: 5
`,
			want: map[string]watcherHttp.DomainPolicy{
				"nordvpn.com": {Max: 10},
				"mullvad.net": {Max: 5},
			},
		},
		{
			name: "list of structs with cooldown",
			body: `run:
  proxy_connection_limits:
    - domain: nordvpn.com
      max: 10
      cooldown_seconds: 10
    - domain: mullvad.net
      max: 5
`,
			want: map[string]watcherHttp.DomainPolicy{
				"nordvpn.com": {Max: 10, Cooldown: 10 * time.Second},
				"mullvad.net": {Max: 5},
			},
		},
		{
			name: "uppercase parent key still resolves via case-insensitive segment match",
			body: `Run:
  proxy_connection_limits:
    - domain: nordvpn.com
      max: 10
    - domain: mullvad.net
      max: 5
`,
			want: map[string]watcherHttp.DomainPolicy{
				"nordvpn.com": {Max: 10},
				"mullvad.net": {Max: 5},
			},
		},
		{
			name: "absent section yields empty map",
			body: `run:
  force: true
`,
			want: map[string]watcherHttp.DomainPolicy{},
		},
		{
			name: "entries with missing or zero values are dropped",
			body: `run:
  proxy_connection_limits:
    - domain: nordvpn.com
      max: 10
    - domain: ""
      max: 99
    - domain: mullvad.net
      max: 0
`,
			want: map[string]watcherHttp.DomainPolicy{"nordvpn.com": {Max: 10}},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			viper.Reset()
			viper.SetConfigType("yaml")
			if err := viper.ReadConfig(bytes.NewBufferString(c.body)); err != nil {
				t.Fatalf("read: %v", err)
			}
			got := loadProxyConnectionLimits()
			if len(got) != len(c.want) {
				t.Fatalf("len mismatch: got %d want %d (%+v vs %+v)", len(got), len(c.want), got, c.want)
			}
			for k, v := range c.want {
				if got[k] != v {
					t.Errorf("key %q: got %+v want %+v", k, got[k], v)
				}
			}
		})
	}
}

// TestProxyLimits_AddUpdateRemove exercises the read/write helpers used by the
// `proxy-limits` CLI subcommands end-to-end against a temp config file,
// verifying the YAML on disk stays in the canonical list-of-structs shape
// through additions, updates, and removals.
func TestProxyLimits_AddUpdateRemove(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".watcher.yaml")
	if err := os.WriteFile(path, []byte("placeholder: 1\n"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	reload := func() map[string]watcherHttp.DomainPolicy {
		viper.Reset()
		viper.SetConfigFile(path)
		if err := viper.ReadInConfig(); err != nil {
			t.Fatalf("reread: %v", err)
		}
		return loadProxyConnectionLimits()
	}

	// add two entries — one with a cooldown, one without
	viper.Reset()
	viper.SetConfigFile(path)
	if err := viper.ReadInConfig(); err != nil {
		t.Fatalf("read: %v", err)
	}
	writeProxyLimitsList([]configuration.ProxyConnectionLimit{
		{Domain: "nordvpn.com", Max: 10, CooldownSeconds: 10},
		{Domain: "mullvad.net", Max: 5},
	})

	got := reload()
	if got["nordvpn.com"] != (watcherHttp.DomainPolicy{Max: 10, Cooldown: 10 * time.Second}) {
		t.Fatalf("after add nordvpn: %+v", got["nordvpn.com"])
	}
	if got["mullvad.net"] != (watcherHttp.DomainPolicy{Max: 5}) {
		t.Fatalf("after add mullvad: %+v", got["mullvad.net"])
	}

	// update nordvpn -> 8, cooldown unchanged at struct level (caller decides)
	entries := readProxyLimitsList()
	for i := range entries {
		if entries[i].Domain == "nordvpn.com" {
			entries[i].Max = 8
		}
	}
	writeProxyLimitsList(entries)

	got = reload()
	if got["nordvpn.com"].Max != 8 || got["nordvpn.com"].Cooldown != 10*time.Second {
		t.Fatalf("after update: %+v", got["nordvpn.com"])
	}

	// remove mullvad
	entries = readProxyLimitsList()
	kept := entries[:0]
	for _, e := range entries {
		if e.Domain != "mullvad.net" {
			kept = append(kept, e)
		}
	}
	writeProxyLimitsList(kept)

	got = reload()
	if _, ok := got["mullvad.net"]; ok {
		t.Fatalf("mullvad.net should have been removed: %+v", got)
	}
	if got["nordvpn.com"].Max != 8 {
		t.Fatalf("nordvpn.com should still be max=8: %+v", got["nordvpn.com"])
	}

	// verify YAML on disk uses the canonical list shape with the cooldown key.
	// (map keys within each item are sorted alphabetically by the YAML writer,
	// so we assert on shape markers rather than line-leading order.)
	body, _ := os.ReadFile(path)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "proxy_connection_limits:") ||
		!strings.Contains(bodyStr, "domain: nordvpn.com") {
		t.Fatalf("YAML not in expected list shape:\n%s", bodyStr)
	}
	if !strings.Contains(bodyStr, "cooldown_seconds: 10") {
		t.Fatalf("cooldown_seconds missing from on-disk YAML:\n%s", bodyStr)
	}
}
