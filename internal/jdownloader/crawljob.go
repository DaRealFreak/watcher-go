// Package jdownloader writes JDownloader Folder Watch ".crawljob" files for
// external links that watcher-go cannot download itself, so JDownloader can
// fetch them straight into the correct per-post folder.
package jdownloader

// Crawljob is one entry of a JDownloader Folder Watch ".crawljob" file
// (a JSON array of these). JDownloader's FolderWatch extension ingests the
// file and adds each package.
//
// The BooleanStatus fields (AutoConfirm/AutoStart/ForcedStart/
// ExtractAfterDownload) use the string enum "TRUE"/"FALSE"/"UNSET". Enabled
// is the JSON string "true".
type Crawljob struct {
	PackageName          string `json:"packageName"`
	DownloadFolder       string `json:"downloadFolder"`
	Comment              string `json:"comment,omitempty"`
	Text                 string `json:"text"`
	Enabled              string `json:"enabled"`
	AutoConfirm          string `json:"autoConfirm"`
	AutoStart            string `json:"autoStart"`
	ForcedStart          string `json:"forcedStart"`
	ExtractAfterDownload string `json:"extractAfterDownload"`
}

// boolStatus maps a Go bool to JDownloader's BooleanStatus string enum.
func boolStatus(b bool) string {
	if b {
		return "TRUE"
	}
	return "FALSE"
}
