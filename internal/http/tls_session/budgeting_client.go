package tls_session

import (
	"context"
	"io"

	watcherHttp "github.com/DaRealFreak/watcher-go/internal/http"
	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
)

// sessionInfoProvider returns the moduleKey and current proxy settings of
// the owning session. The moduleKey lets the budget route requests through
// an active lease if one exists for this module + account.
type sessionInfoProvider func() (moduleKey string, ps *watcherHttp.ProxySettings)

// budgetingTlsClient embeds the underlying tls_client.HttpClient so all
// non-request methods (cookies, SetProxy, dialer, etc.) pass through
// unchanged. Do/Get/Head/Post are intercepted to acquire a budget slot,
// then the response body is wrapped so Close releases it. If the budget is
// nil or the session has no active proxy, the wrapped methods are no-ops
// over the inner client.
type budgetingTlsClient struct {
	tls_client.HttpClient
	info sessionInfoProvider
}

func newBudgetingTlsClient(inner tls_client.HttpClient, fn sessionInfoProvider) tls_client.HttpClient {
	return &budgetingTlsClient{HttpClient: inner, info: fn}
}

func (c *budgetingTlsClient) acquire(ctx context.Context) (*watcherHttp.Slot, error) {
	if watcherHttp.Global == nil {
		return nil, nil
	}
	moduleKey, ps := c.info()
	return watcherHttp.Global.Acquire(ctx, moduleKey, ps)
}

func (c *budgetingTlsClient) bindToBody(resp *http.Response, err error, slot *watcherHttp.Slot) (*http.Response, error) {
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

func (c *budgetingTlsClient) Do(req *http.Request) (*http.Response, error) {
	slot, err := c.acquire(req.Context())
	if err != nil {
		return nil, err
	}
	resp, doErr := c.HttpClient.Do(req)
	return c.bindToBody(resp, doErr, slot)
}

func (c *budgetingTlsClient) Get(url string) (*http.Response, error) {
	slot, err := c.acquire(context.Background())
	if err != nil {
		return nil, err
	}
	resp, getErr := c.HttpClient.Get(url)
	return c.bindToBody(resp, getErr, slot)
}

func (c *budgetingTlsClient) Head(url string) (*http.Response, error) {
	slot, err := c.acquire(context.Background())
	if err != nil {
		return nil, err
	}
	resp, headErr := c.HttpClient.Head(url)
	return c.bindToBody(resp, headErr, slot)
}

func (c *budgetingTlsClient) Post(url, ct string, body io.Reader) (*http.Response, error) {
	slot, err := c.acquire(context.Background())
	if err != nil {
		return nil, err
	}
	resp, postErr := c.HttpClient.Post(url, ct, body)
	return c.bindToBody(resp, postErr, slot)
}
