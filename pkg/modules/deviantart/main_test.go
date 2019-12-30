package deviantart

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	KeyFeed       = "feed"
	KeyGallery    = "gallery"
	KeyAll        = "all"
	KeyCollection = "collection"
	KeyTag        = "tag"
)

func testPattern(t *testing.T, expectedGroup string, pattern *regexp.Regexp) {
	urls := map[string][]string{
		KeyFeed: {
			"https://www.deviantart.com", "https://www.deviantart.com/",
		},
		KeyGallery: {
			"https://www.deviantart.com/test/gallery/12345/test",
			"https://www.deviantart.com/test/gallery/12345/test/",
			"https://www.deviantart.com/test/gallery/12345", "https://www.deviantart.com/test/gallery/12345/",
		},
		KeyAll: {
			"https://www.deviantart.com/test", "https://www.deviantart.com/test/",
			"https://www.deviantart.com/test/gallery", "https://www.deviantart.com/test/gallery/",
			"https://www.deviantart.com/test/gallery/all", "https://www.deviantart.com/test/gallery/all/",
		},
		KeyCollection: {
			"https://www.deviantart.com/testuser/favourites/12345/test-title",
			"https://www.deviantart.com/testuser/favourites/12345/test-title/",
			"https://www.deviantart.com/testuser/favourites/12345",
			"https://www.deviantart.com/testuser/favourites/12345/",
		},
		KeyTag: {
			"https://www.deviantart.com/tag/test", "https://www.deviantart.com/tag/test/",
		},
	}

	for grp, grpURLs := range urls {
		for _, url := range grpURLs {
			assert.New(t).Equal(grp == expectedGroup, pattern.MatchString(url))
		}
	}
}

func TestDeviantArt_Pattern(t *testing.T) {
	patterns := getDeviantArtPattern()
	testPattern(t, KeyFeed, patterns.feedPattern)
	testPattern(t, KeyGallery, patterns.galleryPattern)
	testPattern(t, KeyCollection, patterns.collectionPattern)
	testPattern(t, KeyTag, patterns.tagPattern)
	testPattern(t, KeyAll, patterns.userPattern)
}
