package tapas

import "testing"

func TestExtractEpisodeImages(t *testing.T) {
	body := `
<div class="viewer">
  <img class="content__img js-lazy"
       src="data:image/gif;base64,placeholder"
       data-src="https://us-a.tapas.io/c/86/abc-0.png?__token__=exp%3D1&amp;version=v4">
  <img class="content__img js-lazy"
       src="data:image/gif;base64,placeholder"
       data-src="https://us-a.tapas.io/c/86/abc-1.png?__token__=exp%3D2&amp;version=v4">
  <img class="other-image" src="https://example.com/avatar.png">
</div>
`

	urls, err := extractEpisodeImages(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(urls) != 2 {
		t.Fatalf("expected 2 image urls, got %d: %v", len(urls), urls)
	}

	for _, u := range urls {
		if got, dontWant := u, "&amp;"; got == "" || (containsAmpEntity(got, dontWant)) {
			t.Fatalf("expected decoded URL, got %q", u)
		}
	}
}

func TestBuildPageFileName(t *testing.T) {
	cases := []struct {
		name     string
		index    int
		url      string
		expected string
	}{
		{"first page png", 0, "https://us-a.tapas.io/c/86/abc-0.png?__token__=foo", "001.png"},
		{"jpg extension", 1, "https://us-a.tapas.io/c/86/abc-1.jpg?__token__=foo", "002.jpg"},
		{"unparseable url falls back to .png", 4, ":not a url:", "005.png"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := buildPageFileName(c.index, c.url)
			if got != c.expected {
				t.Fatalf("expected %q, got %q", c.expected, got)
			}
		})
	}
}

// TestBuildPageFileName_NoCollisionAcrossGroups locks in the fix for a bug
// where filenames were derived from the CDN URL's -N suffix. Tapas groups
// multi-image uploads and restarts the suffix at 0 in each group, so two
// images with identical -0 suffixes from different groups produced the same
// filename and one overwrote the other on disk.
func TestBuildPageFileName_NoCollisionAcrossGroups(t *testing.T) {
	urls := []string{
		"https://us-a.tapas.io/c/62/groupA-0.png?__token__=x",
		"https://us-a.tapas.io/c/62/groupA-1.png?__token__=x",
		"https://us-a.tapas.io/c/62/groupA-2.png?__token__=x",
		"https://us-a.tapas.io/c/e3/groupB-0.png?__token__=x",
		"https://us-a.tapas.io/c/e3/groupB-1.png?__token__=x",
	}

	seen := map[string]bool{}
	for i, u := range urls {
		name := buildPageFileName(i, u)
		if seen[name] {
			t.Fatalf("filename collision at index %d (%s) produced duplicate %q", i, u, name)
		}
		seen[name] = true
	}
}

func containsAmpEntity(s, needle string) bool {
	for i := 0; i+len(needle) <= len(s); i++ {
		if s[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
