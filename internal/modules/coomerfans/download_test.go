package coomerfans

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

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
