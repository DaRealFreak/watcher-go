package pawchive

import (
	"net/url"
	"testing"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/pawchive/api"
)

func TestGetDownloadLinks(t *testing.T) {
	m := &pawchive{}
	m.baseUrl, _ = url.Parse("https://pawchive.st")

	post := &api.Post{
		File: api.File{Name: "main.png", Path: "/8a/af/main.png"},
		Attachments: []api.Attachment{
			{Name: "alt1.png", Path: "/91/c9/alt1.png"},
			{Name: "alt2.png", Path: "/1c/c9/alt2.png"},
		},
	}

	links := m.getDownloadLinks(post)
	if len(links) != 3 {
		t.Fatalf("expected 3 links (file + 2 attachments), got %d: %+v", len(links), links)
	}

	want := []struct {
		uri  string
		name string
	}{
		{"https://file.pawchive.st/data/8a/af/main.png?f=main.png", "main.png"},
		{"https://file.pawchive.st/data/91/c9/alt1.png?f=alt1.png", "alt1.png"},
		{"https://file.pawchive.st/data/1c/c9/alt2.png?f=alt2.png", "alt2.png"},
	}
	for i, w := range want {
		if links[i].FileURI != w.uri {
			t.Errorf("links[%d].FileURI = %q, want %q", i, links[i].FileURI, w.uri)
		}
		if links[i].FileName != w.name {
			t.Errorf("links[%d].FileName = %q, want %q", i, links[i].FileName, w.name)
		}
	}
}

func TestGetDownloadLinks_skipsEmptyPaths(t *testing.T) {
	m := &pawchive{}
	m.baseUrl, _ = url.Parse("https://pawchive.st")

	// pawchive posts often have an empty "file": {} — it must not produce a link.
	post := &api.Post{
		File:        api.File{},
		Attachments: []api.Attachment{{Name: "only.png", Path: "/aa/bb/only.png"}},
	}

	links := m.getDownloadLinks(post)
	if len(links) != 1 {
		t.Fatalf("expected 1 link (empty file skipped), got %d", len(links))
	}
	if links[0].FileURI != "https://file.pawchive.st/data/aa/bb/only.png?f=only.png" {
		t.Errorf("FileURI = %q", links[0].FileURI)
	}
}

func TestGetDownloadLinks_skipsMegaIcon(t *testing.T) {
	m := &pawchive{}
	m.baseUrl, _ = url.Parse("https://pawchive.st")
	post := &api.Post{
		Attachments: []api.Attachment{
			{Name: "https://mega.nz/rich-file.png", Path: "/xx/yy/icon.png"},
			{Name: "real.png", Path: "/aa/bb/real.png"},
		},
	}
	links := m.getDownloadLinks(post)
	if len(links) != 1 {
		t.Fatalf("expected mega icon skipped, got %d links: %+v", len(links), links)
	}
	if links[0].FileURI != "https://file.pawchive.st/data/aa/bb/real.png?f=real.png" {
		t.Errorf("FileURI = %q", links[0].FileURI)
	}
}

func TestGetDownloadLinks_inlineImages(t *testing.T) {
	m := &pawchive{}
	m.baseUrl, _ = url.Parse("https://pawchive.st")

	// A relative src is resolved against baseUrl; http:// and https:// absolute
	// srcs must be left untouched (http:// must NOT be prefixed with baseUrl).
	post := &api.Post{
		Content: `<p>` +
			`<img src="/x/y.png">` +
			`<img src="https://cdn.example.com/abs.png">` +
			`<img src="http://cdn.example.com/abs2.png">` +
			`</p>`,
	}

	links := m.getDownloadLinks(post)
	if len(links) != 3 {
		t.Fatalf("expected 3 inline image links, got %d: %+v", len(links), links)
	}

	want := []string{
		"https://pawchive.st/x/y.png",
		"https://cdn.example.com/abs.png",
		"http://cdn.example.com/abs2.png",
	}
	for i, w := range want {
		if links[i].FileURI != w {
			t.Errorf("links[%d].FileURI = %q, want %q", i, links[i].FileURI, w)
		}
	}
}

func TestGetExternalLinks(t *testing.T) {
	m := &pawchive{}
	m.settings.ExternalURLs.PrintExternalItems = true

	post := &api.Post{
		User:    "4829343",
		Content: "grab it here https://mega.nz/folder/abc",
		Embed:   api.Embed{Url: "https://drive.google.com/file/d/xyz"},
	}
	comments := []api.Comment{
		{Commenter: "4829343", Content: "mirror https://mega.nz/file/def"},
		{Commenter: "99999", Content: "spam https://mega.nz/file/notmine"},
	}

	links := m.getExternalLinks(post, comments)

	has := func(want string) bool {
		for _, l := range links {
			if l == want {
				return true
			}
		}
		return false
	}

	if !has("https://drive.google.com/file/d/xyz") {
		t.Errorf("expected embed url, got %v", links)
	}
	if !has("https://mega.nz/folder/abc") {
		t.Errorf("expected content link, got %v", links)
	}
	if !has("https://mega.nz/file/def") {
		t.Errorf("expected creator comment link, got %v", links)
	}
	if has("https://mega.nz/file/notmine") {
		t.Errorf("non-creator comment link must be excluded, got %v", links)
	}
}

func TestGetExternalLinks_disabledBySettings(t *testing.T) {
	m := &pawchive{} // both settings false
	post := &api.Post{Content: "https://mega.nz/folder/abc", Embed: api.Embed{Url: "https://drive.google.com/x"}}

	if links := m.getExternalLinks(post, nil); len(links) != 0 {
		t.Errorf("expected no links when external_urls disabled, got %v", links)
	}
}

func TestExtractDataPath(t *testing.T) {
	cases := []struct {
		uri  string
		want string
	}{
		{"https://file.pawchive.st/data/b3/9d/abc.jpg", "b3/9d/abc.jpg"},
		// pawchive download URLs append a ?f={name}; the query must be stripped.
		{"https://file.pawchive.st/data/b3/9d/abc.jpg?f=Isaka+0.jpg", "b3/9d/abc.jpg"},
		{"https://img.pawchive.st/thumbnail/data/cc/dd/x.jpg", "cc/dd/x.jpg"},
		{"https://cdn.example.com/abs.png", ""},
		{"", ""},
	}
	for _, tc := range cases {
		if got := extractDataPath(tc.uri); got != tc.want {
			t.Errorf("extractDataPath(%q) = %q, want %q", tc.uri, got, tc.want)
		}
	}
}

func TestIsImageFile(t *testing.T) {
	cases := map[string]bool{
		"foo.jpg":  true,
		"foo.JPG":  true,
		"foo.jpeg": true,
		"foo.png":  true,
		"foo.gif":  true,
		"foo.webp": true,
		"foo.bmp":  true,
		"foo.mp4":  false,
		"foo.zip":  false,
		"foo":      false,
		"":         false,
	}
	for name, want := range cases {
		if got := isImageFile(name); got != want {
			t.Errorf("isImageFile(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestBuildThumbnailURL(t *testing.T) {
	m := &pawchive{}

	t.Run("image file maps to img host thumbnail (query stripped)", func(t *testing.T) {
		item := &models.DownloadQueueItem{
			FileURI: "https://file.pawchive.st/data/b3/9d/abc.jpg?f=Isaka+0.jpg",
		}
		got := m.buildThumbnailURL(item, "Isaka 0.jpg")
		want := "https://img.pawchive.st/thumbnail/data/b3/9d/abc.jpg"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("falls back to hashed-path extension when name has none", func(t *testing.T) {
		item := &models.DownloadQueueItem{FileURI: "https://file.pawchive.st/data/aa/bb/img.png"}
		got := m.buildThumbnailURL(item, "")
		want := "https://img.pawchive.st/thumbnail/data/aa/bb/img.png"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("non-image returns empty", func(t *testing.T) {
		item := &models.DownloadQueueItem{FileURI: "https://file.pawchive.st/data/aa/bb/clip.mp4"}
		if got := m.buildThumbnailURL(item, "clip.mp4"); got != "" {
			t.Errorf("non-image should return empty, got %q", got)
		}
	})

	t.Run("non-data url returns empty", func(t *testing.T) {
		item := &models.DownloadQueueItem{FileURI: "https://cdn.example.com/abs.png"}
		if got := m.buildThumbnailURL(item, "abs.png"); got != "" {
			t.Errorf("url without /data/ should return empty, got %q", got)
		}
	})
}

// TestFileDownloadTarget is the regression test for the reported bug: an
// un-archived post (has_full=false) must never trigger a full-res request - the
// file host returns 504 (not 404) for those, which previously fataled the parse.
// Instead, images resolve straight to the thumbnail and non-image files are
// skipped, while archived posts and external (non-/data/) URLs download directly.
func TestFileDownloadTarget(t *testing.T) {
	m := &pawchive{}
	m.baseUrl, _ = url.Parse("https://pawchive.st")

	const (
		imageURI    = "https://file.pawchive.st/data/ed/54/abc.png?f=page.png"
		imageThumb  = "https://img.pawchive.st/thumbnail/data/ed/54/abc.png"
		nonImageURI = "https://file.pawchive.st/data/2b/05/bundle.bin?f=ch14.rar"
		externalURI = "https://cdn.example.com/inline.png"
	)

	cases := []struct {
		name          string
		hasFull       bool
		item          *models.DownloadQueueItem
		fileName      string
		wantURL       string
		wantThumbnail bool
		wantSkip      bool
	}{
		{
			name: "archived image downloads full-res directly",
			hasFull: true, item: &models.DownloadQueueItem{FileURI: imageURI}, fileName: "page.png",
			wantURL: imageURI,
		},
		{
			name: "un-archived image falls back to thumbnail (no full-res request)",
			hasFull: false, item: &models.DownloadQueueItem{FileURI: imageURI}, fileName: "page.png",
			wantURL: imageThumb, wantThumbnail: true,
		},
		{
			name: "un-archived non-image (.rar) is skipped",
			hasFull: false, item: &models.DownloadQueueItem{FileURI: nonImageURI}, fileName: "ch14.rar",
			wantSkip: true,
		},
		{
			name: "archived non-image downloads full-res directly",
			hasFull: true, item: &models.DownloadQueueItem{FileURI: nonImageURI}, fileName: "ch14.rar",
			wantURL: nonImageURI,
		},
		{
			name: "external inline image downloads directly even when un-archived",
			hasFull: false, item: &models.DownloadQueueItem{FileURI: externalURI}, fileName: "inline.png",
			wantURL: externalURI,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			post := &api.Post{HasFull: tc.hasFull}
			gotURL, gotThumb, gotSkip := m.fileDownloadTarget(post, tc.item, tc.fileName)
			if gotSkip != tc.wantSkip {
				t.Fatalf("skip = %v, want %v", gotSkip, tc.wantSkip)
			}
			if tc.wantSkip {
				return
			}
			if gotURL != tc.wantURL {
				t.Errorf("url = %q, want %q", gotURL, tc.wantURL)
			}
			if gotThumb != tc.wantThumbnail {
				t.Errorf("isThumbnail = %v, want %v", gotThumb, tc.wantThumbnail)
			}
		})
	}
}

func TestGetSubFolder(t *testing.T) {
	m := &pawchive{}
	item := &models.TrackedItem{URI: "https://pawchive.st/patreon/user/4829343"}
	if got := m.getSubFolder(item); got != "patreon/4829343" {
		t.Errorf("getSubFolder = %q, want %q", got, "patreon/4829343")
	}
}
