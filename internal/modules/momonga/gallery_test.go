package momonga

import (
	"fmt"
	"testing"
)

func newTestModule() *momonga {
	return NewBareModule().ModuleInterface.(*momonga)
}

func TestNewBareModule(t *testing.T) {
	module := NewBareModule()

	if module.Key != "momon-ga.com" {
		t.Fatalf("unexpected module key: %q", module.Key)
	}
	if len(module.URISchemas) != 1 {
		t.Fatalf("expected exactly one URI schema, got %d", len(module.URISchemas))
	}
	if !module.URISchemas[0].MatchString("https://momon-ga.com/fanzine/mo3959358/") {
		t.Error("URI schema should match a gallery URL")
	}
	if !module.URISchemas[0].MatchString("https://momon-ga.com/tag/full-color/") {
		t.Error("URI schema should match a listing URL")
	}
}

func TestGalleryPattern(t *testing.T) {
	m := newTestModule()

	cases := map[string]bool{
		"https://momon-ga.com/fanzine/mo3959358/":  true,
		"https://momon-ga.com/magazine/mo3960630/": true,
		"https://momon-ga.com/tag/full-color/":     false,
		"https://momon-ga.com/cartoonist/foo/":     false,
	}
	for uri, want := range cases {
		if got := m.galleryPattern.MatchString(uri); got != want {
			t.Errorf("galleryPattern.MatchString(%q) = %v, want %v", uri, got, want)
		}
	}

	matches := m.galleryPattern.FindStringSubmatch("https://momon-ga.com/fanzine/mo3959358/")
	if len(matches) != 2 || matches[1] != "3959358" {
		t.Errorf("expected gallery id 3959358, got %#v", matches)
	}
}

const galleryHTMLFixture = `
<html><body>
<h1>Test Gallery Title</h1>
<div id="post-number">3ページ</div>
<div id="post-tag">
  <div class="post-tags"><a href="/group/foo/" rel="tag">チョフ畑</a></div>
  <div class="post-tags">
    <a href="/tag/full-color/" rel="tag">full color</a>
    <a href="/tag/mind-control/" rel="tag">mind control</a>
  </div>
</div>
<div id="post-hentai">
  <span id="more-1"></span>
  <img src="https://z3.momon-ga.com/galleries/3959358/1.webp" />
  <img src="https://z3.momon-ga.com/galleries/3959358/2.webp" />
  <img src="https://z3.momon-ga.com/galleries/3959358/3.webp" />
</div>
<div class="post-list-image">
  <img src="https://z2.momon-ga.com/galleries/1814174/1.webp" alt="related, must be ignored" />
</div>
</body></html>`

func TestExtractGalleryTitle(t *testing.T) {
	m := newTestModule()
	if got := m.extractGalleryTitle(galleryHTMLFixture); got != "Test Gallery Title" {
		t.Errorf("extractGalleryTitle = %q, want %q", got, "Test Gallery Title")
	}
}

func TestExtractGalleryImages(t *testing.T) {
	m := newTestModule()
	images := m.extractGalleryImages(galleryHTMLFixture)

	if len(images) != 3 {
		t.Fatalf("expected 3 images (related image must be ignored), got %d", len(images))
	}
	for i, img := range images {
		wantPage := i + 1
		if img.page != wantPage {
			t.Errorf("image %d: page = %d, want %d", i, img.page, wantPage)
		}
		wantURI := fmt.Sprintf("https://z3.momon-ga.com/galleries/3959358/%d.webp", wantPage)
		if img.uri != wantURI {
			t.Errorf("image %d: uri = %q, want %q", i, img.uri, wantURI)
		}
	}
}

func TestExtractContentTags(t *testing.T) {
	m := newTestModule()
	tags := m.extractContentTags(galleryHTMLFixture)
	if len(tags) != 3 {
		t.Fatalf("expected 3 tags, got %d (%#v)", len(tags), tags)
	}
}

func TestIsBlacklisted(t *testing.T) {
	m := newTestModule()

	if matched, _ := m.isBlacklisted(galleryHTMLFixture, "Test Gallery Title"); matched {
		t.Error("expected not blacklisted when no tags configured")
	}

	m.settings.Search.BlacklistedTags = []string{"mind control"}
	if matched, term := m.isBlacklisted(galleryHTMLFixture, "Test Gallery Title"); !matched || term != "mind control" {
		t.Errorf("expected blacklist match on content tag, got matched=%v term=%q", matched, term)
	}

	m.settings.Search.BlacklistedTags = []string{"GALLERY"}
	if matched, _ := m.isBlacklisted(galleryHTMLFixture, "Test Gallery Title"); !matched {
		t.Error("expected case-insensitive blacklist match on title")
	}
}
