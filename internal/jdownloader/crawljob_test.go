package jdownloader

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCrawljobJSONEncoding(t *testing.T) {
	job := Crawljob{
		PackageName:          "pawchive.st - 123 - Title",
		DownloadFolder:       `C:\Downloads\x`,
		Comment:              "https://pawchive.st/p/123",
		Text:                 "https://mega.nz/file/a\nhttps://mediafire.com/b",
		Enabled:              "true",
		AutoConfirm:          boolStatus(true),
		AutoStart:            boolStatus(false),
		ForcedStart:          "UNSET",
		ExtractAfterDownload: "UNSET",
	}

	b, err := json.Marshal([]Crawljob{job})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)

	for _, want := range []string{
		`"packageName":"pawchive.st - 123 - Title"`,
		`"downloadFolder":"C:\\Downloads\\x"`,
		`"enabled":"true"`,
		`"autoConfirm":"TRUE"`,
		`"autoStart":"FALSE"`,
		`"forcedStart":"UNSET"`,
		`"extractAfterDownload":"UNSET"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("encoded crawljob missing %q\ngot: %s", want, s)
		}
	}
}

func TestBoolStatus(t *testing.T) {
	if boolStatus(true) != "TRUE" {
		t.Errorf("boolStatus(true) = %q", boolStatus(true))
	}
	if boolStatus(false) != "FALSE" {
		t.Errorf("boolStatus(false) = %q", boolStatus(false))
	}
}
