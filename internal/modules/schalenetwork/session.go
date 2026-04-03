package schalenetwork

import (
	http "github.com/bogdanfinn/fhttp"
)

func (m *schaleNetwork) get(uri string) (*http.Response, error) {
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}

	return m.do(req)
}

func (m *schaleNetwork) post(uri string) (*http.Response, error) {
	req, err := http.NewRequest("POST", uri, nil)
	if err != nil {
		return nil, err
	}

	return m.do(req)
}

func (m *schaleNetwork) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Referer", "https://niyaniya.moe/")
	req.Header.Set("Origin", "https://niyaniya.moe")

	return m.Session.Do(req)
}
