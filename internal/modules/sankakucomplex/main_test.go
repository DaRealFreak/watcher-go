package sankakucomplex

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// matchesAnySchema reports whether any of the module's registered URI schemas
// match the passed value.
func matchesAnySchema(value string) bool {
	for _, schema := range NewBareModule().URISchemas {
		if schema.MatchString(value) {
			return true
		}
	}

	return false
}

// TestURISchemaMatchesTagOverview locks in that the /tags/{tag} overview URL is
// selectable by the module so it can be added and parsed at all.
func TestURISchemaMatchesTagOverview(t *testing.T) {
	assert.New(t).True(matchesAnySchema("https://www.sankakucomplex.com/tags/test"))
}

// TestExtractTagFromOverviewURI guards the /tags/{tag} endpoint contract: the
// tag must be read from the URL path, and the post/book search URLs that the
// overview fans out to (which carry the tag in a "tags" query parameter) must
// NOT be detected as overview URIs again - otherwise parseTagAggregate would
// recurse into itself indefinitely.
func TestExtractTagFromOverviewURI(t *testing.T) {
	cases := []struct {
		name        string
		uri         string
		expectedTag string
		expectedOk  bool
	}{
		{"plain tag overview", "https://www.sankakucomplex.com/tags/test", "test", true},
		{"trailing slash", "https://www.sankakucomplex.com/tags/test/", "test", true},
		{"locale prefix", "https://www.sankakucomplex.com/en/tags/test", "test", true},
		{"url encoded tag", "https://www.sankakucomplex.com/tags/some%20tag", "some tag", true},
		{"gallery search is not an overview", "https://www.sankakucomplex.com/?tags=test", "", false},
		{"book search is not an overview", "https://www.sankakucomplex.com/books?tags=test", "", false},
		{"single post is not an overview", "https://www.sankakucomplex.com/post/show/12345", "", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tagName, ok := extractTagFromOverviewURI(c.uri)
			assert.New(t).Equal(c.expectedOk, ok)
			assert.New(t).Equal(c.expectedTag, tagName)
		})
	}
}
