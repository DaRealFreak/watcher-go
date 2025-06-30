package xpff

import (
	"encoding/json"
	"net/url"
	"time"
)

type Handler struct {
	generator XPFFHeaderGenerator
	guestID   string
	userAgent string
}

type Content struct {
	NavigatorProperties NavigatorProperties `json:"navigator_properties"`
	CreatedAt           int64               `json:"created_at"`
}

type NavigatorProperties struct {
	HasBeenActive string `json:"hasBeenActive"`
	UserAgent     string `json:"userAgent"`
	Webdriver     string `json:"webdriver"`
}

func NewHandler(guestID string, userAgent string) *Handler {
	baseKey := "0e6be1f1e21ffc33590b888fd4dc81b19713e570e805d4e5df80a493c9571a05"
	generator := XPFFHeaderGenerator{baseKey: baseKey}

	return &Handler{
		generator: generator,
		guestID:   guestID,
		userAgent: userAgent,
	}
}

func (h *Handler) GetXPFFHeader() (string, error) {
	content := Content{
		NavigatorProperties: NavigatorProperties{
			HasBeenActive: "true",
			UserAgent:     h.userAgent,
			Webdriver:     "false",
		},
		CreatedAt: time.Now().UnixMilli(),
	}

	contentJSON, err := json.Marshal(content)
	if err != nil {
		return "", err
	}

	return h.generator.GenerateXPFF(string(contentJSON), url.QueryEscape(h.guestID))
}
