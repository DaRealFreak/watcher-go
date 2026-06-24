package fourchan

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/http/tls_session"
	fhttp "github.com/bogdanfinn/fhttp"
)

// proxyKey returns a stable identifier for a loop proxy used to track eviction.
func proxyKey(p *http.ProxySettings) string {
	return fmt.Sprintf("%s:%d", p.Host, p.Port)
}

// nextLiveProxyIndex returns the index of the next enabled, non-evicted loop proxy
// after the current ProxyLoopIndex, wrapping around to the start. It returns -1 when no
// usable proxy remains. Evicted proxies are those that returned a 403 earlier in this
// run (see getPage).
func (m *fourChan) nextLiveProxyIndex() int {
	count := len(m.settings.LoopProxies)
	for offset := 1; offset <= count; offset++ {
		idx := (m.ProxyLoopIndex + offset) % count
		proxy := &m.settings.LoopProxies[idx]
		if proxy.Enable && !m.evictedProxies[proxyKey(proxy)] {
			return idx
		}
	}

	return -1
}

// hasLiveLoopProxy reports whether at least one enabled, non-evicted loop proxy remains.
func (m *fourChan) hasLiveLoopProxy() bool {
	for i := range m.settings.LoopProxies {
		proxy := &m.settings.LoopProxies[i]
		if proxy.Enable && !m.evictedProxies[proxyKey(proxy)] {
			return true
		}
	}

	return false
}

// evictCurrentProxy marks the proxy at the current loop index as dead for the rest of
// the run so future rotations skip it.
func (m *fourChan) evictCurrentProxy() {
	if m.ProxyLoopIndex < 0 || m.ProxyLoopIndex >= len(m.settings.LoopProxies) {
		return
	}

	if m.evictedProxies == nil {
		m.evictedProxies = make(map[string]bool)
	}

	m.evictedProxies[proxyKey(&m.settings.LoopProxies[m.ProxyLoopIndex])] = true
}

// getPage fetches a page, rotating through the loop-proxy pool. Each call advances to
// the next live proxy (which also resets that session's rate limiter). When a proxy
// responds with 403 it is evicted from rotation for the rest of the run and the request
// is retried on the next live proxy. Once every proxy has been evicted the 403 is
// returned so the caller skips the item. When loop mode is disabled it falls back to a
// plain session Get, leaving the existing single-proxy/direct behavior unchanged.
//
// The page-retrieval eviction set is independent of the multi-proxy download pool: a
// proxy blocked by the archive search may still serve image downloads, so they evict
// separately.
func (m *fourChan) getPage(uri string) (*fhttp.Response, error) {
	if !m.settings.Loop {
		return m.Session.Get(uri)
	}

	for {
		if err := m.setProxyMethod(); err != nil {
			return nil, err
		}

		res, err := m.Session.Get(uri)
		if err == nil {
			return res, nil
		}

		// only a 403 triggers eviction; every other error is propagated as-is
		var statusErr tls_session.StatusError
		if !errors.As(err, &statusErr) || statusErr.StatusCode != 403 {
			return res, err
		}

		evictedHost := m.settings.LoopProxies[m.ProxyLoopIndex].Host
		m.evictCurrentProxy()

		if !m.hasLiveLoopProxy() {
			slog.Warn(fmt.Sprintf(
				"proxy \"%s\" returned 403 and no usable proxies remain, skipping uri: %s",
				evictedHost, uri), "module", m.Key)

			return res, err
		}

		slog.Warn(fmt.Sprintf(
			"proxy \"%s\" returned 403, evicting it and retrying with another proxy for uri: %s",
			evictedHost, uri), "module", m.Key)
	}
}
