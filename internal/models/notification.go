package models

import log "github.com/sirupsen/logrus"

// Notification contains information for the download queue to notify the user about
type Notification struct {
	Message string    `json:"message"`
	Level   log.Level `json:"level"`
}
