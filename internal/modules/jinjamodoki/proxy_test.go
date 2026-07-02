package jinjamodoki

import (
	"testing"

	"github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/http/tls_session"
	"github.com/stretchr/testify/assert"
)

// newLoopModule returns a jinjaModoki module wired for proxy-loop testing. The session
// is real but offline: SetProxy only reconfigures the transport, it never dials.
func newLoopModule(proxies []http.ProxySettings) *jinjaModoki {
	m := NewBareModule().ModuleInterface.(*jinjaModoki)
	m.Session = tls_session.NewTlsClientSession(m.Key)
	m.settings.Loop = true
	m.settings.LoopProxies = proxies

	return m
}

func proxy(host string, port int, enabled bool) http.ProxySettings {
	return http.ProxySettings{Enable: enabled, Host: host, Port: port, Type: "https"}
}

// TestSetProxyMethod_LoopSkipsDisabled is the regression test for the "index out of
// range [-1]" panic in the proxy loop: the old skip loop reset ProxyLoopIndex to -1
// (or ran it past the slice) when a disabled proxy sat at the rotation boundary, then
// crashed on the slice access. Rotation must skip disabled proxies, only select enabled
// ones, and wrap around without panicking.
func TestSetProxyMethod_LoopSkipsDisabled(t *testing.T) {
	tests := []struct {
		name         string
		proxies      []http.ProxySettings
		enabledHosts map[string]bool
	}{
		{
			name:         "first disabled",
			proxies:      []http.ProxySettings{proxy("a", 1, false), proxy("b", 2, true)},
			enabledHosts: map[string]bool{"b": true},
		},
		{
			name:         "last disabled",
			proxies:      []http.ProxySettings{proxy("a", 1, true), proxy("b", 2, true), proxy("c", 3, false)},
			enabledHosts: map[string]bool{"a": true, "b": true},
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

			assert.Len(t, seen, len(tt.enabledHosts), "rotation must visit every enabled proxy")
		})
	}
}

// TestSetProxyMethod_LoopNoEnabledProxies verifies that when every loop proxy is
// disabled the rotation returns a descriptive error instead of panicking.
func TestSetProxyMethod_LoopNoEnabledProxies(t *testing.T) {
	m := newLoopModule([]http.ProxySettings{proxy("a", 1, false), proxy("b", 2, false)})

	var err error
	assert.NotPanics(t, func() {
		err = m.setProxyMethod()
	})
	assert.Error(t, err, "no usable loop proxies must surface an error")
}
