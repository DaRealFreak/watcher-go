package deviantart

import (
	"testing"

	"github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/modules/deviantart/napi"
	"github.com/stretchr/testify/assert"
)

// newLoopModule returns a deviantArt module wired for proxy-loop testing. The nAPI
// session is real but offline: SetProxy only reconfigures the transport, it never dials.
func newLoopModule(proxies []http.ProxySettings) *deviantArt {
	m := NewBareModule().ModuleInterface.(*deviantArt)
	m.nAPI = napi.NewDeviantartNAPI(m.Key, "", nil)
	m.settings.Loop = true
	m.settings.LoopProxies = proxies

	return m
}

func proxy(host string, port int, enabled bool) http.ProxySettings {
	return http.ProxySettings{Enable: enabled, Host: host, Port: port, Type: "https"}
}

// TestSetProxyMethod_LoopSkipsDisabled is the regression test for the live panic
// "index out of range [-1]" in the deviantart proxy loop. The old hand-rolled skip
// loop reset ProxyLoopIndex to -1 (or ran it past the slice) when it reached the end
// while skipping a disabled proxy, then dereferenced LoopProxies[ProxyLoopIndex] and
// crashed. Rotation must skip disabled proxies, only ever select enabled ones, and
// wrap around without panicking.
func TestSetProxyMethod_LoopSkipsDisabled(t *testing.T) {
	tests := []struct {
		name    string
		proxies []http.ProxySettings
		// enabledHosts is the set of hosts a rotation is allowed to select.
		enabledHosts map[string]bool
	}{
		{
			// exact shape that triggered the live "index out of range [-1]" panic:
			// the first proxy is disabled, so the skip loop walked to the end and
			// reset the index to -1 before indexing the slice.
			name:         "first disabled",
			proxies:      []http.ProxySettings{proxy("a", 1, false), proxy("b", 2, true)},
			enabledHosts: map[string]bool{"b": true},
		},
		{
			// last proxy disabled reproduces the sibling positive out-of-range case:
			// the skip loop ran the index past the end of the slice.
			name:         "last disabled",
			proxies:      []http.ProxySettings{proxy("a", 1, true), proxy("b", 2, true), proxy("c", 3, false)},
			enabledHosts: map[string]bool{"a": true, "b": true},
		},
		{
			name:         "middle disabled",
			proxies:      []http.ProxySettings{proxy("a", 1, true), proxy("b", 2, false), proxy("c", 3, true)},
			enabledHosts: map[string]bool{"a": true, "c": true},
		},
		{
			name:         "all enabled",
			proxies:      []http.ProxySettings{proxy("a", 1, true), proxy("b", 2, true)},
			enabledHosts: map[string]bool{"a": true, "b": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newLoopModule(tt.proxies)

			seen := map[string]bool{}
			// run through more than a full cycle so wrap-around is exercised.
			for i := 0; i < len(tt.proxies)*2+1; i++ {
				var err error
				assert.NotPanics(t, func() {
					err = m.setProxyMethod()
				}, "setProxyMethod must not panic while rotating proxies")
				assert.NoError(t, err, "a rotation with at least one enabled proxy must not error")

				host := m.settings.LoopProxies[m.ProxyLoopIndex].Host
				assert.Truef(t, tt.enabledHosts[host], "rotation selected disabled proxy %q", host)
				seen[host] = true
			}

			// every enabled proxy must be reachable over a full cycle.
			assert.Len(t, seen, len(tt.enabledHosts), "rotation must visit every enabled proxy")
		})
	}
}

// TestSetProxyMethod_LoopNoEnabledProxies verifies that when every loop proxy is
// disabled the rotation returns a descriptive error instead of panicking or silently
// applying an invalid proxy.
func TestSetProxyMethod_LoopNoEnabledProxies(t *testing.T) {
	m := newLoopModule([]http.ProxySettings{proxy("a", 1, false), proxy("b", 2, false)})

	var err error
	assert.NotPanics(t, func() {
		err = m.setProxyMethod()
	})
	assert.Error(t, err, "no usable loop proxies must surface an error")
}
