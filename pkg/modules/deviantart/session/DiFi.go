package session

// diFiResponse is the same response as the default OAuth2 application API call
type diFiResponse struct {
	Status  string      `json:"status"`
	Content interface{} `json:"content"`
}

// diFiRequest is mirroring the sent request
type diFiRequest struct {
	Class  string      `json:"class"`
	Method string      `json:"method"`
	Args   interface{} `json:"args"`
}

// diFiCall is the response for each API call
type diFiCall struct {
	Request  *diFiRequest  `json:"request"`
	Response *diFiResponse `json:"response"`
}

// diFiCallCollection contains all API function call responses
type diFiCallCollection struct {
	Calls []*diFiCall `json:"calls"`
}

// DiFi is the parent response for the calls
type DiFi struct {
	Status   string             `json:"status"`
	Response diFiCallCollection `json:"response"`
}

// DeveloperConsoleResponse is the struct of the developer console response
type DeveloperConsoleResponse struct {
	DiFi `json:"DiFi"`
}
