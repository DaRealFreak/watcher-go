package coomerfans

import "testing"

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
