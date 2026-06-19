package api

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// JSON below is reduced from the real tag-and-wiki responses; the alias-relevant
// shape (alias_of as a populated object vs an empty object {}) is preserved.
func TestTagAndWikiResponse_AliasTarget(t *testing.T) {
	cases := []struct {
		name          string
		json          string
		wantCanonical string
		wantAliased   bool
	}{
		{
			name:          "aliased tag resolves to canonical tagName",
			json:          `{"tag":{"name":"centaurea_shianus","tagName":"centaurea_shianus","post_count":0,"alias_of":{"tagName":"centorea_shianus","name":"Centorea Shianus"}}}`,
			wantCanonical: "centorea_shianus",
			wantAliased:   true,
		},
		{
			name:          "canonical tag with empty alias_of object is not aliased",
			json:          `{"tag":{"name":"huge_breasts","tagName":"huge_breasts","post_count":3731969,"alias_of":{},"alias_tags":[{"tagName":"huge_boobs"},{"tagName":"huge_tits"}]}}`,
			wantCanonical: "",
			wantAliased:   false,
		},
		{
			name:          "tag aliased to itself is not aliased",
			json:          `{"tag":{"name":"foo","tagName":"foo","alias_of":{"tagName":"foo","name":"Foo"}}}`,
			wantCanonical: "",
			wantAliased:   false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var resp TagAndWikiResponse
			require.NoError(t, json.Unmarshal([]byte(c.json), &resp))

			canonical, aliased := resp.AliasTarget()
			assert.Equal(t, c.wantAliased, aliased)
			assert.Equal(t, c.wantCanonical, canonical)
		})
	}
}
