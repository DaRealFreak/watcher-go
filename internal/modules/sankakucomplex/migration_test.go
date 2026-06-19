package sankakucomplex

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsQualifierTag(t *testing.T) {
	assert.True(t, isQualifierTag("pool:123"))
	assert.True(t, isQualifierTag("order:date"))
	assert.False(t, isQualifierTag("centaurea_shianus"))
}

func TestSplitSearchTags(t *testing.T) {
	assert.Equal(t, []string{"a", "b"}, splitSearchTags("a b"))
	assert.Equal(t, []string{"a"}, splitSearchTags("  a  "))
	assert.Empty(t, splitSearchTags(""))
}

func TestRebuildSearchTags(t *testing.T) {
	resolve := func(tag string) (string, bool) {
		if tag == "centaurea_shianus" {
			return "centorea_shianus", true
		}
		return tag, false
	}

	t.Run("single aliased tag", func(t *testing.T) {
		rebuilt, changed := rebuildSearchTags([]string{"centaurea_shianus"}, resolve)
		assert.True(t, changed)
		assert.Equal(t, "centorea_shianus", rebuilt)
	})

	t.Run("multi-tag with one aliased", func(t *testing.T) {
		rebuilt, changed := rebuildSearchTags([]string{"centaurea_shianus", "huge_breasts"}, resolve)
		assert.True(t, changed)
		assert.Equal(t, "centorea_shianus huge_breasts", rebuilt)
	})

	t.Run("qualifier tokens are left untouched", func(t *testing.T) {
		rebuilt, changed := rebuildSearchTags([]string{"order:date", "centaurea_shianus"}, resolve)
		assert.True(t, changed)
		assert.Equal(t, "order:date centorea_shianus", rebuilt)
	})

	t.Run("nothing aliased reports no change", func(t *testing.T) {
		rebuilt, changed := rebuildSearchTags([]string{"huge_breasts"}, resolve)
		assert.False(t, changed)
		assert.Equal(t, "huge_breasts", rebuilt)
	})
}

func TestFollowAliasChain(t *testing.T) {
	chain := func(m map[string]string) func(string) (string, bool, error) {
		return func(tag string) (string, bool, error) {
			next, ok := m[tag]
			return next, ok, nil
		}
	}

	t.Run("not aliased", func(t *testing.T) {
		final, changed, err := followAliasChain("huge_breasts", chain(map[string]string{}))
		require.NoError(t, err)
		assert.False(t, changed)
		assert.Equal(t, "huge_breasts", final)
	})

	t.Run("single hop", func(t *testing.T) {
		final, changed, err := followAliasChain("a", chain(map[string]string{"a": "b"}))
		require.NoError(t, err)
		assert.True(t, changed)
		assert.Equal(t, "b", final)
	})

	t.Run("multi hop", func(t *testing.T) {
		final, changed, err := followAliasChain("a", chain(map[string]string{"a": "b", "b": "c"}))
		require.NoError(t, err)
		assert.True(t, changed)
		assert.Equal(t, "c", final)
	})

	t.Run("cycle terminates", func(t *testing.T) {
		// x -> y -> x: start "x" is pre-visited, hop to "y", then "x" is already
		// visited so the loop breaks deterministically with current == "y".
		final, changed, err := followAliasChain("x", chain(map[string]string{"x": "y", "y": "x"}))
		require.NoError(t, err)
		assert.True(t, changed)
		assert.Equal(t, "y", final)
	})

	t.Run("lookup error stops without migrating", func(t *testing.T) {
		lookup := func(string) (string, bool, error) { return "", false, assert.AnError }
		final, changed, err := followAliasChain("a", lookup)
		require.Error(t, err)
		assert.False(t, changed)
		assert.Equal(t, "a", final)
	})
}

func TestMigrateTagsInURI(t *testing.T) {
	t.Run("query form single tag", func(t *testing.T) {
		got, err := migrateTagsInURI("https://www.sankakucomplex.com/?tags=centaurea_shianus&tab=explore", "centorea_shianus")
		require.NoError(t, err)

		parsed, perr := url.Parse(got)
		require.NoError(t, perr)
		assert.Equal(t, "centorea_shianus", parsed.Query().Get("tags"))
		assert.Equal(t, "explore", parsed.Query().Get("tab")) // preserved here; AddItem drops it later
	})

	t.Run("query form multi tag", func(t *testing.T) {
		got, err := migrateTagsInURI("https://www.sankakucomplex.com/?tags=centaurea_shianus+huge_breasts", "centorea_shianus huge_breasts")
		require.NoError(t, err)

		parsed, perr := url.Parse(got)
		require.NoError(t, perr)
		assert.Equal(t, "centorea_shianus huge_breasts", parsed.Query().Get("tags"))
	})

	t.Run("books query form", func(t *testing.T) {
		got, err := migrateTagsInURI("https://www.sankakucomplex.com/books?tags=centaurea_shianus", "centorea_shianus")
		require.NoError(t, err)

		parsed, perr := url.Parse(got)
		require.NoError(t, perr)
		assert.Equal(t, "/books", parsed.Path)
		assert.Equal(t, "centorea_shianus", parsed.Query().Get("tags"))
	})

	t.Run("tags overview path form", func(t *testing.T) {
		got, err := migrateTagsInURI("https://www.sankakucomplex.com/tags/centaurea_shianus", "centorea_shianus")
		require.NoError(t, err)
		assert.Equal(t, "https://www.sankakucomplex.com/tags/centorea_shianus", got)
	})
}
