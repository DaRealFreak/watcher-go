package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	watcherHttp "github.com/DaRealFreak/watcher-go/internal/http"
)

const ChounyuuDomain = "chounyuu.com"
const ChounyuuApiVersion = 1

const SuperFutaDomain = "superfuta.com"
const SuperFutaApiVersion = 2

// ChounyuuAPI contains all required items to communicate with the API
type ChounyuuAPI struct {
	Session watcherHttp.SessionInterface
}

// mapAPIResponse maps the API response into the passed APIResponse type
func (a *ChounyuuAPI) mapAPIResponse(res *http.Response, apiRes interface{}) (err error) {
	out, err := ioutil.ReadAll(res.Body)
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
