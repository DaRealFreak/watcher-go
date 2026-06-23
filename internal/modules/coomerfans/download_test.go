package coomerfans

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

// TestPostRelPathPreservesSubFolderSeparator guards against re-flattening the
// "{service}/{username}" subfolder. A prior version re-ran fp.SanitizePath with
// allowSeparator=false on the already-built subfolder, collapsing the "/" to "_"
// (producing "onlyfans_zayafterhouz" instead of the nested "onlyfans/zayafterhouz"
// the design specifies).
func TestPostRelPathPreservesSubFolderSeparator(t *testing.T) {
	got := filepath.ToSlash(postRelPath("onlyfans/zayafterhouz", "68811621", "Full video 6 min", "2892ac-image.jpg", 0))
	if !strings.HasPrefix(got, "onlyfans/zayafterhouz/") {
		t.Errorf("subfolder nesting lost: got %q, want prefix \"onlyfans/zayafterhouz/\"", got)
	}
	if strings.Contains(got, "onlyfans_zayafterhouz") {
		t.Errorf("subfolder separator flattened to underscore: %q", got)
	}
	if !strings.Contains(got, "/68811621 - Full video 6 min/68811621_1_2892ac-image.jpg") {
		t.Errorf("unexpected post-folder/filename layout: %q", got)
	}
}

// TestPostRelPathEmptyTitle verifies the post folder is just the ID when the
// title is empty (the single-post tracked-item path, which carries no title).
func TestPostRelPathEmptyTitle(t *testing.T) {
	got := filepath.ToSlash(postRelPath("onlyfans/zayafterhouz", "68811621", "", "clip.mp4", 1))
	if !strings.HasSuffix(got, "/68811621/68811621_2_clip.mp4") {
		t.Errorf("empty-title post folder should be just the id: %q", got)
	}
}

func postDoc(t *testing.T) *goquery.Document {
	t.Helper()
	data, err := os.ReadFile("testdata/post.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	return doc
}

func TestExtractPostMedia(t *testing.T) {
	media := extractPostMedia(postDoc(t))
	if len(media) != 2 {
		t.Fatalf("got %d media urls, want 2 (image + video source)", len(media))
	}
	for _, u := range media {
		if strings.Contains(u, "/istorage/") {
			t.Errorf("avatar url leaked into media: %s", u)
		}
		if !strings.Contains(u, "img1.coomerfans.com/storage/") {
			t.Errorf("unexpected media host: %s", u)
		}
	}
	if !strings.HasSuffix(media[0], "image.jpg") {
		t.Errorf("media[0] = %q, want the content image first", media[0])
	}
	if !strings.Contains(media[1], "video.mp4?e=") {
		t.Errorf("media[1] = %q, want the signed video source", media[1])
	}
}

func TestScrapeUsername(t *testing.T) {
	if got := scrapeUsername(postDoc(t)); got != "zayafterhouz" {
		t.Errorf("scrapeUsername = %q, want zayafterhouz", got)
	}
}
