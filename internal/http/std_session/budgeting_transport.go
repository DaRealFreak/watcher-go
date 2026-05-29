package std_session

import (
	"net/http"

	watcherHttp "github.com/DaRealFreak/watcher-go/internal/http"
)

// sessionInfoProvider returns the moduleKey and current proxy settings of
// the owning session. The moduleKey lets the budget route requests through
// an active lease if one exists for this module + account.
type sessionInfoProvider func() (moduleKey string, ps *watcherHttp.ProxySettings)

// budgetingTransport wraps an inner RoundTripper to gate every request on
// the global ConnectionBudget. If the budget is nil or the session has no
// active proxy, RoundTrip is a no-op pass-through. On a successful response
// the body is wrapped so Close releases the slot.
type budgetingTransport struct {
	inner http.RoundTripper
	info  sessionInfoProvider
}

func newBudgetingTransport(inner http.RoundTripper, fn sessionInfoProvider) http.RoundTripper {
	if inner == nil {
		inner = http.DefaultTransport
	}
	return &budgetingTransport{inner: inner, info: fn}
}

func (t *budgetingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var slot *watcherHttp.Slot
	if watcherHttp.Global != nil {
		moduleKey, ps := t.info()
		s, err := watcherHttp.Global.Acquire(req.Context(), moduleKey, ps)
		if err != nil {
			return nil, err
		}
		slot = s
	}
	resp, err := t.inner.RoundTrip(req)
	if slot == nil {
		return resp, err
	}
	if err != nil || resp == nil || resp.Body == nil {
		slot.Release()
		return resp, err
	}
	resp.Body = watcherHttp.WrapBodyWithSlot(resp.Body, slot)
	return resp, err
}

// unwrapBudgetingTransport returns the original inner transport if rt is a
// budgetingTransport - used by SetProxy to clone the underlying http.Transport
// without nesting wrappers on every reset.
func unwrapBudgetingTransport(rt http.RoundTripper) http.RoundTripper {
	if bt, ok := rt.(*budgetingTransport); ok {
		return bt.inner
	}
	return rt
}
