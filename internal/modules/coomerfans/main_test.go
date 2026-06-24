package coomerfans

import "testing"

func TestNewBareModule(t *testing.T) {
	m := NewBareModule()
	if m.ModuleKey() != "coomerfans.com" {
		t.Fatalf("ModuleKey = %q, want coomerfans.com", m.ModuleKey())
	}
	urls := []string{
		"https://coomerfans.com/u/onlyfans/324235/zayafterhouz",
		"https://coomerfans.com/p/68811621/324235/onlyfans",
	}
	for _, u := range urls {
		matched := false
		for _, schema := range m.URISchemas {
			if schema.MatchString(u) {
				matched = true
				break
			}
		}
		if !matched {
			t.Errorf("no URI schema matched %q", u)
		}
	}
}
