package api

import "testing"

func TestParseEpisodeListBody(t *testing.T) {
	// Trimmed copy of the fragment tapas.io returns from /series/{id}/episodes.
	// Locks in the live class names so a regression in title extraction (logs
	// showed "downloading episode 12345 ()" because the wrong selector was
	// used) is caught immediately.
	body := `
<li class="js-tiara-tracking body__item" data-href="/episode/123" data-id="123" id="ep-123">
  <a class="item__thumb"><img src="thumb.png"/></a>
  <div class="item__info">
    <a class="info__label">Episode 1</a>
    <a class="info__title">Ep. 1</a>
  </div>
</li>
<li class="js-tiara-tracking body__item" data-href="/episode/456" data-id="456" id="ep-456">
  <a class="item__thumb"><img src="thumb.png"/></a>
  <div class="item__info">
    <a class="info__label">Episode 2</a>
    <a class="info__title">Ep. 2</a>
  </div>
</li>
<li class="not-an-episode">noise</li>
<li data-id="not-numeric">should be skipped</li>
`

	items, err := parseEpisodeListBody(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 episodes, got %d", len(items))
	}

	if items[0].ID != "123" || items[0].Title != "Ep. 1" {
		t.Fatalf("first episode mismatch: %+v", items[0])
	}
	if items[1].ID != "456" || items[1].Title != "Ep. 2" {
		t.Fatalf("second episode mismatch: %+v", items[1])
	}
}

func TestCleanSeriesTitle(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		expected string
	}{
		{"web community suffix", "Read My Childhood Friends Are Both 10 Meters Tall | Tapas Web Community", "My Childhood Friends Are Both 10 Meters Tall"},
		{"generic tapas suffix", "Read Something | Tapas Web Novel", "Something"},
		{"no suffix", "Plain Title", "Plain Title"},
		{"no read prefix", "Foo | Tapas Web Community", "Foo"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := cleanSeriesTitle(c.in)
			if got != c.expected {
				t.Fatalf("expected %q, got %q", c.expected, got)
			}
		})
	}
}
