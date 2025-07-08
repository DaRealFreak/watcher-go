package http

import (
	"fmt"
	"net/url"
	"strings"
)

// ProxySettings are the proxy server settings for the session
type ProxySettings struct {
	Enable   bool   `mapstructure:"enable"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Type     string `mapstructure:"type"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

func (s ProxySettings) GetProxyType() string {
	switch strings.ToUpper(s.Type) {
	case "SOCKS5":
		return "socks5"
	case "HTTP":
		return "http"
	case "HTTPS", "":
		return "https"
	default:
		return s.Type
	}
}

func (s ProxySettings) GetProxyString() string {
	var authString string
	if s.Username != "" && s.Password != "" {
		authString = fmt.Sprintf("%s:%s@", url.QueryEscape(s.Username), url.QueryEscape(s.Password))
	}

	return fmt.Sprintf(
		"%s://%s%s:%d",
		s.GetProxyType(),
		authString,
		url.QueryEscape(s.Host), s.Port,
	)
}
