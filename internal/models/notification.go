package models

import "log/slog"

// Notification contains information for the download queue to notify the user about
type Notification struct {
	Message string     `json:"message"`
	Level   slog.Level `json:"level"`
}
