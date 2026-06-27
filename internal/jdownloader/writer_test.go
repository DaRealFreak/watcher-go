package jdownloader

import "testing"

func TestBlacklisted(t *testing.T) {
	w := NewWriter(Config{Blacklist: []string{"mediafire.com", "Discord.gg", "t.me"}})

	cases := map[string]bool{
		"https://www.mediafire.com/file/abc":  true,  // subdomain match
		"https://mediafire.com/file/abc":      true,  // exact host match
		"http://cdn.discord.gg/x":             true,  // case-insensitive blacklist entry
		"https://t.me/somechannel":            true,
		"https://mega.nz/file/abc":            false, // not listed
		"https://notmediafire.com/file/abc":   false, // must not match by substring
		"mega.nz/file/abc":                    false, // scheme-less, not listed
		"":                                    false,
	}
	for raw, want := range cases {
		if got := w.Blacklisted(raw); got != want {
			t.Errorf("Blacklisted(%q) = %v, want %v", raw, got, want)
		}
	}
}

func TestBlacklistedSchemeless(t *testing.T) {
	w := NewWriter(Config{Blacklist: []string{"mediafire.com"}})
	if !w.Blacklisted("www.mediafire.com/file/abc") {
		t.Errorf("scheme-less blacklisted host should still match")
	}
}
