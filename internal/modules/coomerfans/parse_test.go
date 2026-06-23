package coomerfans

import (
	"bytes"
	"os"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestParsePostURL(t *testing.T) {
	cases := []struct {
		uri                        string
		wantID, wantUser, wantSvc  string
		wantOK                     bool
	}{
		{"https://coomerfans.com/p/68811621/324235/onlyfans", "68811621", "324235", "onlyfans", true},
		{"/p/123/456/fansly", "123", "456", "fansly", true},
		{"https://coomerfans.com/u/onlyfans/324235/zayafterhouz", "", "", "", false},
		{"https://coomerfans.com/random-post", "", "", "", false},
	}
	for _, c := range cases {
		id, user, svc, ok := parsePostURL(c.uri)
		if ok != c.wantOK || id != c.wantID || user != c.wantUser || svc != c.wantSvc {
			t.Errorf("parsePostURL(%q) = (%q,%q,%q,%v), want (%q,%q,%q,%v)",
				c.uri, id, user, svc, ok, c.wantID, c.wantUser, c.wantSvc, c.wantOK)
		}
	}
}

func TestParseUserURL(t *testing.T) {
	cases := []struct {
		uri                            string
		wantSvc, wantUser, wantName    string
		wantOK                         bool
	}{
		{"https://coomerfans.com/u/onlyfans/324235/zayafterhouz", "onlyfans", "324235", "zayafterhouz", true},
		{"https://coomerfans.com/u/fansly/346538/thedeloreangirl", "fansly", "346538", "thedeloreangirl", true},
		{"/u/onlyfans/389377/hazel.baby.g", "onlyfans", "389377", "hazel.baby.g", true},
		{"https://coomerfans.com/p/68811621/324235/onlyfans", "", "", "", false},
	}
	for _, c := range cases {
		svc, user, name, ok := parseUserURL(c.uri)
		if ok != c.wantOK || svc != c.wantSvc || user != c.wantUser || name != c.wantName {
			t.Errorf("parseUserURL(%q) = (%q,%q,%q,%v), want (%q,%q,%q,%v)",
				c.uri, svc, user, name, ok, c.wantSvc, c.wantUser, c.wantName, c.wantOK)
		}
	}
}

func docFromFixture(t *testing.T, name string) *goquery.Document {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("parse fixture %s: %v", name, err)
	}
	return doc
}

func TestExtractPostRefs(t *testing.T) {
	refs := extractPostRefs(docFromFixture(t, "creator.html"))
	if len(refs) != 3 {
		t.Fatalf("got %d refs, want 3 (avatar links and duplicate View Post links must be excluded)", len(refs))
	}
	if refs[0].ID != "68811621" || refs[0].UserID != "324235" || refs[0].Service != "onlyfans" {
		t.Errorf("ref[0] = %+v, want ID 68811621 / user 324235 / onlyfans", refs[0])
	}
	if refs[0].Title != "Full video 6 min" {
		t.Errorf("ref[0].Title = %q, want %q", refs[0].Title, "Full video 6 min")
	}
	if refs[2].ID != "68811655" {
		t.Errorf("ref[2].ID = %q, want 68811655 (document order preserved)", refs[2].ID)
	}
}

func TestExtractPostRefsEmpty(t *testing.T) {
	refs := extractPostRefs(docFromFixture(t, "creator_empty.html"))
	if len(refs) != 0 {
		t.Fatalf("got %d refs, want 0 for an out-of-range page", len(refs))
	}
}

func TestSubFolderForURI(t *testing.T) {
	cases := []struct {
		uri  string
		want string
	}{
		{"https://coomerfans.com/u/onlyfans/324235/zayafterhouz", "onlyfans/zayafterhouz"},
		{"https://coomerfans.com/u/fansly/346538/thedeloreangirl", "fansly/thedeloreangirl"},
		{"https://coomerfans.com/p/68811621/324235/onlyfans", ""},
	}
	for _, c := range cases {
		if got := subFolderForURI(c.uri); got != c.want {
			t.Errorf("subFolderForURI(%q) = %q, want %q", c.uri, got, c.want)
		}
	}
}
