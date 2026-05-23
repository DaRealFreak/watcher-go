package kemono

import (
	"errors"
	"net/url"
	"testing"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/kemono/api"
)

func TestJoinDataURL(t *testing.T) {
	cases := []struct {
		name string
		host string
		path string
		want string
	}{
		{
			name: "leading slash path - the bug we are fixing",
			host: "https://n4.kemono.cr",
			path: "/bc/55/bc55effd88ae74f39017df84e793434ead5e33c70c7f9f2aabcfe09ba90eb846.jpg",
			want: "https://n4.kemono.cr/data/bc/55/bc55effd88ae74f39017df84e793434ead5e33c70c7f9f2aabcfe09ba90eb846.jpg",
		},
		{
			name: "no leading slash path",
			host: "https://kemono.cr",
			path: "bc/55/file.jpg",
			want: "https://kemono.cr/data/bc/55/file.jpg",
		},
		{
			name: "trailing slash host",
			host: "https://n1.kemono.cr/",
			path: "/aa/bb/file.jpg",
			want: "https://n1.kemono.cr/data/aa/bb/file.jpg",
		},
		{
			name: "double-leading-slash path",
			host: "https://kemono.cr",
			path: "//aa/bb/file.jpg",
			want: "https://kemono.cr/data/aa/bb/file.jpg",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := joinDataURL(tc.host, tc.path)
			if got != tc.want {
				t.Errorf("joinDataURL(%q, %q) = %q, want %q", tc.host, tc.path, got, tc.want)
			}
		})
	}
}

func TestIsNetworkError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"unrelated", errors.New("some random error"), false},
		{"windows connectex", errors.New(`Get "https://n4.kemono.cr/data/...": dial tcp [2a0a:cd80:1001:91::]:443: connectex: A connection attempt failed because the connected party did not properly respond after a period of time`), true},
		{"i/o timeout", errors.New("dial tcp 1.2.3.4:443: i/o timeout"), true},
		{"connection refused", errors.New("dial tcp 1.2.3.4:443: connection refused"), true},
		{"connection reset", errors.New("read tcp: connection reset by peer"), true},
		{"no such host", errors.New("dial tcp: lookup nN.kemono.cr: no such host"), true},
		{"TLS handshake timeout", errors.New("net/http: TLS handshake timeout"), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isNetworkError(tc.err); got != tc.want {
				t.Errorf("isNetworkError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestGetDownloadLinks_attachmentsFromPostObject(t *testing.T) {
	m := &kemono{}
	m.baseUrl, _ = url.Parse("https://kemono.cr")

	srvN3 := "https://n3.kemono.cr"
	srvN4 := "https://n4.kemono.cr"

	root := &api.PostRoot{
		Post: api.Post{
			File: api.File{Name: "main.jpg", Path: "/bd/86/main.jpg"},
			Attachments: []api.Attachment{
				{Name: "a1.jpg", Path: "/2d/72/a1.jpg"},
				{Name: "a2.jpg", Path: "/49/4a/a2.jpg"},
			},
		},
		Attachments: []api.Attachment{},
		Previews: []api.Thumbnail{
			{Type: "thumbnail", Name: "main.jpg", Path: "/bd/86/main.jpg", Server: &srvN3},
			{Type: "thumbnail", Name: "a1.jpg", Path: "/2d/72/a1.jpg", Server: &srvN4},
			{Type: "thumbnail", Name: "a2.jpg", Path: "/49/4a/a2.jpg", Server: &srvN3},
		},
	}

	links := m.getDownloadLinks(root)
	if len(links) != 3 {
		t.Fatalf("expected 3 links (main + 2 attachments), got %d: %+v", len(links), links)
	}

	want := []struct {
		fileURI  string
		fallback string
		name     string
	}{
		{"https://n3.kemono.cr/data/bd/86/main.jpg", "https://kemono.cr/data/bd/86/main.jpg", "main.jpg"},
		{"https://n4.kemono.cr/data/2d/72/a1.jpg", "https://kemono.cr/data/2d/72/a1.jpg", "a1.jpg"},
		{"https://n3.kemono.cr/data/49/4a/a2.jpg", "https://kemono.cr/data/49/4a/a2.jpg", "a2.jpg"},
	}
	for i, w := range want {
		if links[i].FileURI != w.fileURI {
			t.Errorf("links[%d].FileURI = %q, want %q", i, links[i].FileURI, w.fileURI)
		}
		if links[i].FallbackFileURI != w.fallback {
			t.Errorf("links[%d].FallbackFileURI = %q, want %q", i, links[i].FallbackFileURI, w.fallback)
		}
		if links[i].FileName != w.name {
			t.Errorf("links[%d].FileName = %q, want %q", i, links[i].FileName, w.name)
		}
	}
}

func TestGetDownloadLinks_previewWithoutServerUsesMainHost(t *testing.T) {
	m := &kemono{}
	m.baseUrl, _ = url.Parse("https://kemono.cr")

	root := &api.PostRoot{
		Post: api.Post{
			File: api.File{Name: "main.jpg", Path: "/bd/86/main.jpg"},
		},
		Previews: []api.Thumbnail{
			{Type: "thumbnail", Name: "main.jpg", Path: "/bd/86/main.jpg"},
		},
	}

	links := m.getDownloadLinks(root)
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}
	if links[0].FileURI != "https://kemono.cr/data/bd/86/main.jpg" {
		t.Errorf("FileURI = %q, want main-host URL", links[0].FileURI)
	}
	if links[0].FallbackFileURI != "" {
		t.Errorf("FallbackFileURI should be empty when no CDN host known, got %q", links[0].FallbackFileURI)
	}
}

func TestGetDownloadLinks_unmatchedPreviewBecomesStandalone(t *testing.T) {
	m := &kemono{}
	m.baseUrl, _ = url.Parse("https://kemono.cr")

	srv := "https://n2.kemono.cr"
	root := &api.PostRoot{
		Post: api.Post{
			File: api.File{Name: "main.jpg", Path: "/bd/86/main.jpg"},
		},
		Previews: []api.Thumbnail{
			{Type: "thumbnail", Name: "main.jpg", Path: "/bd/86/main.jpg", Server: &srv},
			{Type: "thumbnail", Name: "extra.png", Path: "/ee/ff/extra.png", Server: &srv},
		},
	}

	links := m.getDownloadLinks(root)
	if len(links) != 2 {
		t.Fatalf("expected 2 links (main + standalone preview), got %d", len(links))
	}
	if links[1].FileURI != "https://n2.kemono.cr/data/ee/ff/extra.png" {
		t.Errorf("standalone preview FileURI = %q", links[1].FileURI)
	}
	if links[1].FallbackFileURI != "https://kemono.cr/data/ee/ff/extra.png" {
		t.Errorf("standalone preview FallbackFileURI = %q", links[1].FallbackFileURI)
	}
}

func TestExtractDataPath(t *testing.T) {
	cases := []struct {
		uri  string
		want string
	}{
		{"https://n4.kemono.cr/data/bc/55/file.jpg", "bc/55/file.jpg"},
		{"https://kemono.cr/data/aa/bb/file.png", "aa/bb/file.png"},
		{"https://img.kemono.cr/thumbnail/data/cc/dd/x.jpg", "cc/dd/x.jpg"},
		{"https://kemono.cr/static/foo.svg", ""},
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
		"foo.jpg":                              true,
		"foo.JPG":                              true,
		"foo.jpeg":                             true,
		"foo.png":                              true,
		"foo.gif":                              true,
		"foo.webp":                             true,
		"foo.bmp":                              true,
		"foo.mp4":                              false,
		"foo.zip":                              false,
		"foo":                                  false,
		"":                                     false,
		"https://n4.kemono.cr/data/aa/bb.jpg":  true,
		"https://n4.kemono.cr/data/aa/bb.mp4":  false,
	}
	for name, want := range cases {
		if got := isImageFile(name); got != want {
			t.Errorf("isImageFile(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestThumbnailHost(t *testing.T) {
	mKemono := &kemono{}
	mKemono.baseUrl, _ = url.Parse("https://kemono.cr")
	if got := mKemono.thumbnailHost(); got != "img.kemono.cr" {
		t.Errorf("kemono thumbnailHost = %q, want img.kemono.cr", got)
	}

	mCoomer := &kemono{}
	mCoomer.baseUrl, _ = url.Parse("https://coomer.st")
	if got := mCoomer.thumbnailHost(); got != "img.coomer.st" {
		t.Errorf("coomer thumbnailHost = %q, want img.coomer.st", got)
	}
}

func TestBuildThumbnailURL(t *testing.T) {
	m := &kemono{}
	m.baseUrl, _ = url.Parse("https://kemono.cr")

	t.Run("derives from FallbackFileURI", func(t *testing.T) {
		item := &models.DownloadQueueItem{
			FileURI:         "https://n4.kemono.cr/data/bc/55/x.jpg",
			FallbackFileURI: "https://kemono.cr/data/bc/55/x.jpg",
		}
		got := m.buildThumbnailURL(item, "original-name.jpeg")
		want := "https://img.kemono.cr/thumbnail/data/bc/55/x.jpg"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("derives from FileURI when no fallback", func(t *testing.T) {
		item := &models.DownloadQueueItem{
			FileURI: "https://kemono.cr/data/aa/bb/y.png",
		}
		got := m.buildThumbnailURL(item, "y.png")
		want := "https://img.kemono.cr/thumbnail/data/aa/bb/y.png"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("returns empty for non-image", func(t *testing.T) {
		item := &models.DownloadQueueItem{
			FileURI:         "https://n4.kemono.cr/data/aa/bb/movie.mp4",
			FallbackFileURI: "https://kemono.cr/data/aa/bb/movie.mp4",
		}
		if got := m.buildThumbnailURL(item, "movie.mp4"); got != "" {
			t.Errorf("non-image should return empty, got %q", got)
		}
	})

	t.Run("uses coomer host for coomer base url", func(t *testing.T) {
		mc := &kemono{}
		mc.baseUrl, _ = url.Parse("https://coomer.st")
		item := &models.DownloadQueueItem{
			FileURI: "https://coomer.st/data/ee/ff/img.jpg",
		}
		got := mc.buildThumbnailURL(item, "img.jpg")
		want := "https://img.coomer.st/thumbnail/data/ee/ff/img.jpg"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestGetDownloadLinks_skipsMegaFolderIcons(t *testing.T) {
	m := &kemono{}
	m.baseUrl, _ = url.Parse("https://kemono.cr")

	root := &api.PostRoot{
		Previews: []api.Thumbnail{
			{Type: "thumbnail", Name: "https://mega.nz/rich-file.png", Path: "/xx/yy/rich.png"},
		},
	}

	links := m.getDownloadLinks(root)
	if len(links) != 0 {
		t.Fatalf("expected 0 links (mega icon skipped), got %d", len(links))
	}
}
