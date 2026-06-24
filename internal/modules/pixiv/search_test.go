package pixiv

import (
	"testing"

	mobileapi "github.com/DaRealFreak/watcher-go/internal/modules/pixiv/mobile_api"
)

// newPixivModule returns the registered pixiv module implementation for pattern/mode tests.
func newPixivModule() *pixiv {
	return NewBareModule().ModuleInterface.(*pixiv)
}

// TestSearchPatternModernSearchURL ensures the modern /search?q= URL form is recognized as a
// search (and not mis-routed or dropped to the "could not be associated" default branch).
func TestSearchPatternModernSearchURL(t *testing.T) {
	m := newPixivModule()
	uri := "https://www.pixiv.net/search?q=test&s_mode=tag&type=artwork"

	if !m.patterns.searchPattern.MatchString(uri) {
		t.Fatalf("search pattern did not match modern search URL %q", uri)
	}

	// the Parse switch reaches parseSearch only if none of these claim the URL first
	if m.patterns.fanboxPattern.MatchString(uri) {
		t.Errorf("fanbox pattern unexpectedly matched %q", uri)
	}
	if m.patterns.illustrationPattern.MatchString(uri) {
		t.Errorf("illustration pattern unexpectedly matched %q", uri)
	}
	if m.patterns.memberPattern.MatchString(uri) {
		t.Errorf("member pattern unexpectedly matched %q", uri)
	}

	if word := m.patterns.searchPattern.FindStringSubmatch(uri)[1]; word != "test" {
		t.Errorf("expected extracted search word %q, got %q", "test", word)
	}
}

// TestSearchPatternQueryNotFirst ensures the q parameter is extracted even when it is not the
// first query parameter.
func TestSearchPatternQueryNotFirst(t *testing.T) {
	m := newPixivModule()
	uri := "https://www.pixiv.net/search?s_mode=tag&q=hello%20world&type=artwork"

	match := m.patterns.searchPattern.FindStringSubmatch(uri)
	if match == nil {
		t.Fatalf("search pattern did not match %q", uri)
	}
	if match[1] != "hello%20world" {
		t.Errorf("expected raw search word %q, got %q", "hello%20world", match[1])
	}
}

func TestGetSearchModeFromURI(t *testing.T) {
	m := newPixivModule()

	cases := []struct {
		name    string
		uri     string
		want    string
		wantErr bool
	}{
		// modern /search?q= s_mode keys (from the pixiv web UI selector)
		{"modern tag", "https://www.pixiv.net/search?q=t&s_mode=tag", mobileapi.SearchModePartialTagMatch, false},
		{"modern tag_full", "https://www.pixiv.net/search?q=t&s_mode=tag_full", mobileapi.SearchModeExactTagMatch, false},
		{"modern tc", "https://www.pixiv.net/search?q=t&s_mode=tc", mobileapi.SearchModeTitleAndCaption, false},
		{"modern tag_tc unsupported", "https://www.pixiv.net/search?q=t&s_mode=tag_tc", "", true},
		// legacy /search.php?word= s_mode keys must keep working
		{"legacy s_tag", "https://www.pixiv.net/search.php?word=t&s_mode=s_tag", mobileapi.SearchModePartialTagMatch, false},
		{"legacy s_tag_full", "https://www.pixiv.net/search.php?word=t&s_mode=s_tag_full", mobileapi.SearchModeExactTagMatch, false},
		{"legacy s_tc", "https://www.pixiv.net/search.php?word=t&s_mode=s_tc", mobileapi.SearchModeTitleAndCaption, false},
		{"legacy default empty", "https://www.pixiv.net/search.php?word=t", mobileapi.SearchModeExactTagMatch, false},
		{"unknown mode", "https://www.pixiv.net/search?q=t&s_mode=bogus", "", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := m.getSearchModeFromURI(tc.uri)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got mode %q", tc.uri, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.uri, err)
			}
			if got != tc.want {
				t.Errorf("getSearchModeFromURI(%q) = %q, want %q", tc.uri, got, tc.want)
			}
		})
	}
}
