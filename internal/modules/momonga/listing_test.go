package momonga

import (
	"testing"

	"github.com/DaRealFreak/watcher-go/internal/models"
)

const listingHTMLFixture = `
<html><body>
<div class="post-list">
  <a href="https://momon-ga.com/fanzine/mo3959358/"><div class="post-list-image"></div></a>
  <a href="https://momon-ga.com/fanzine/mo3959358/"><span>duplicate link, same id</span></a>
  <a href="https://momon-ga.com/magazine/mo3960630/"><div class="post-list-image"></div></a>
  <a href="https://momon-ga.com/tag/full-color/">not a gallery, must be ignored</a>
</div>
<div class="page-h">
  <a class="page larger" href="https://momon-ga.com/tag/full-color/page/2/">2</a>
  <a class="nextpostslink" rel="next" href="https://momon-ga.com/tag/full-color/page/2/">&rsaquo;</a>
  <a class="last" href="https://momon-ga.com/tag/full-color/page/1155/">last</a>
</div>
</body></html>`

const listingLastPageFixture = `
<html><body>
<div class="post-list">
  <a href="https://momon-ga.com/fanzine/mo1000001/"><div class="post-list-image"></div></a>
</div>
<div class="page-h"><span class="current">1155</span></div>
</body></html>`

func TestExtractListingGalleries(t *testing.T) {
	m := newTestModule()
	items := m.extractListingGalleries(listingHTMLFixture)

	if len(items) != 2 {
		t.Fatalf("expected 2 deduped galleries, got %d (%#v)", len(items), items)
	}
	if items[0].id != "3959358" || items[0].uri != "https://momon-ga.com/fanzine/mo3959358/" {
		t.Errorf("unexpected first item: %#v", items[0])
	}
	if items[1].id != "3960630" {
		t.Errorf("unexpected second item id: %q", items[1].id)
	}
}

func TestGetNextListingPageURL(t *testing.T) {
	m := newTestModule()

	url, exists := m.getNextListingPageURL(listingHTMLFixture)
	if !exists || url != "https://momon-ga.com/tag/full-color/page/2/" {
		t.Errorf("getNextListingPageURL = (%q, %v), want next page url", url, exists)
	}

	if _, exists := m.getNextListingPageURL(listingLastPageFixture); exists {
		t.Error("expected no next page on the last page")
	}
}

func TestGetSubFolder(t *testing.T) {
	m := newTestModule()

	if got := m.getSubFolder(&models.TrackedItem{URI: "https://momon-ga.com/tag/full-color/"}); got != "" {
		t.Errorf("expected empty subfolder when categorize disabled, got %q", got)
	}

	m.settings.Search.CategorizeSearch = true

	cases := map[string]string{
		"https://momon-ga.com/tag/full-color/":                         "tag full-color",
		"https://momon-ga.com/cartoonist/foo/":                         "cartoonist foo",
		"https://momon-ga.com/group/bar/":                              "group bar",
		"https://momon-ga.com/parody/fate-grand-order/":                "parody fate-grand-order",
		"https://momon-ga.com/trend/":                                  "trend",
		"https://momon-ga.com/popularity/":                             "popularity",
		"https://momon-ga.com/cartoonist/%E3%81%82%E3%81%84%E3%81%86/": "cartoonist あいう",
	}
	for uri, want := range cases {
		if got := m.getSubFolder(&models.TrackedItem{URI: uri}); got != want {
			t.Errorf("getSubFolder(%q) = %q, want %q", uri, got, want)
		}
	}

	m.settings.Search.InheritSubFolder = true
	got := m.getSubFolder(&models.TrackedItem{URI: "https://momon-ga.com/tag/full-color/", SubFolder: "existing"})
	if got != "existing" {
		t.Errorf("expected inherited subfolder, got %q", got)
	}
}
