package api

import (
	"encoding/json"
	http "github.com/bogdanfinn/fhttp"
	"io"
	"strings"

	watcherHttp "github.com/DaRealFreak/watcher-go/internal/http"
)

// ChounyuuDomain is the domain name including the top level domain for chounyuu
const ChounyuuDomain = "chounyuu.com"

// ChounyuuApiVersion is the used API version required for the API access
const ChounyuuApiVersion = 1

// SuperFutaDomain is the domain name including the top level domain for SuperFuta
const SuperFutaDomain = "superfuta.com"

// SuperFutaApiVersion is the used API version required for the API access
const SuperFutaApiVersion = 2

// ChounyuuAPI contains all required items to communicate with the API
type ChounyuuAPI struct {
	Session watcherHttp.TlsClientSessionInterface
}

// mapAPIResponse maps the API response into the passed APIResponse type
func (a *ChounyuuAPI) mapAPIResponse(res *http.Response, apiRes interface{}) (err error) {
	out, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	content := string(out)

	// unmarshal the request content into the response struct
	if err := json.Unmarshal([]byte(content), &apiRes); err != nil {
		return err
	}

	return nil
}

func (a *ChounyuuAPI) getApiVersion(domain string) int {
	if strings.Contains(domain, SuperFutaDomain) {
		return SuperFutaApiVersion
	}
	return ChounyuuApiVersion
}
