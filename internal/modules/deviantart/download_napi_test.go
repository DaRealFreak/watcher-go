package deviantart

import (
	"testing"

	"github.com/DaRealFreak/watcher-go/internal/modules/deviantart/napi"
	"github.com/stretchr/testify/assert"
)

// TestGetTextContent_WithheldMarkup reproduces the root cause of the live
// "unexpected end of JSON input" error.
//
// DeviantArt withholds the body markup for subscription/tier-locked deviations
// ("tierAccess":"locked"): the html type is still reported (e.g. "tiptap") but the
// "markup" field is omitted, so it decodes to an empty string. Parsing an empty
// string as draft/tiptap JSON fails with "unexpected end of JSON input". Missing
// markup means "no accessible text", not a parse failure, so it must decode to
// empty content without an error.
func TestGetTextContent_WithheldMarkup(t *testing.T) {
	for _, htmlType := range []string{"tiptap", "draft", "writer", ""} {
		tc := &napi.TextContent{}
		tc.Html.Type = htmlType
		// markup intentionally left empty (withheld by the server)

		text, err := tc.GetTextContent()
		assert.NoErrorf(t, err, "withheld %q markup must not be a parse error", htmlType)
		assert.Emptyf(t, text, "withheld %q markup must yield empty text", htmlType)
	}
}

// TestDeviationIsTierLocked verifies the definitive tier-lock detection: DeviantArt
// only sets "tierAccess":"locked" on deviations behind a subscription tier the account
// cannot access, and omits the field otherwise.
func TestDeviationIsTierLocked(t *testing.T) {
	locked := &napi.Deviation{
		TierAccess:  napi.TierAccessLocked,
		PrimaryTier: &napi.Tier{Title: "Appreciators"},
	}

	assert.True(t, locked.IsTierLocked())
	assert.Equal(t, "Appreciators", locked.TierName())

	assert.False(t, (&napi.Deviation{}).IsTierLocked(), "absent tierAccess must not be locked")
	assert.False(t, (&napi.Deviation{TierAccess: "granted"}).IsTierLocked())
	assert.Empty(t, (&napi.Deviation{}).TierName())
}

// TestDownloadLiteratureNapi_WithheldMarkup is the regression test for the live bug:
// a tier-locked literature deviation whose body markup is withheld must not abort the
// download with "unexpected end of JSON input" - in multi-proxy mode a returned error
// stops the entire gallery and permanently stalls the collection on this item.
func TestDownloadLiteratureNapi_WithheldMarkup(t *testing.T) {
	m := NewBareModule().ModuleInterface.(*deviantArt)

	tc := &napi.TextContent{}
	tc.Html.Type = "tiptap" // DA reports the type but withholds markup when tier-locked

	dev := &napi.Deviation{
		DeviationId: "925532053",
		Type:        "literature",
		Author:      &napi.Author{Username: "NRawkk"},
		TextContent: tc,
		TierAccess:  napi.TierAccessLocked,
		PrimaryTier: &napi.Tier{Title: "Appreciators"},
	}

	item := downloadQueueItemNAPI{deviation: dev}

	var files []string
	err := m.downloadLiteratureNapi(item, &files)

	assert.NoError(t, err, "tier-locked literature with withheld markup must not abort the download")
	assert.Empty(t, files, "no literature file should be written when the body is withheld")
	assert.True(t, dev.IsTierLocked(), "the deviation must be definitively recognised as tier-locked")
}

// TestDownloadDescriptionNapi_WithheldMarkup ensures the description path also treats
// withheld (empty) markup as "no description" instead of a parse error.
func TestDownloadDescriptionNapi_WithheldMarkup(t *testing.T) {
	m := NewBareModule().ModuleInterface.(*deviantArt)

	desc := &napi.TextContent{}
	desc.Html.Type = "tiptap"
	// markup intentionally left empty (withheld by the server)

	item := downloadQueueItemNAPI{
		deviation: &napi.Deviation{
			DeviationId: "925532053",
			Author:      &napi.Author{Username: "NRawkk"},
			Extended:    &napi.Extended{DescriptionText: desc},
		},
	}

	var files []string
	err := m.downloadDescriptionNapi(item, &files)

	assert.NoError(t, err, "withheld description markup must not abort the download")
	assert.Empty(t, files, "no description file should be written for withheld markup")
}

// TestDownloadDescriptionNapi_CorruptMarkup ensures a deviation whose description markup
// is genuinely corrupt JSON does not abort the download. DeviantArt sometimes mangles
// emoji surrogates in the description markup and swallows a string's closing quote,
// yielding invalid JSON ("invalid character 't' after object key:value pair"). The
// description is best-effort metadata, so a DA-corrupted one must be skipped (only the
// description) rather than aborting the whole deviation - the image still downloads.
func TestDownloadDescriptionNapi_CorruptMarkup(t *testing.T) {
	m := NewBareModule().ModuleInterface.(*deviantArt)

	desc := &napi.TextContent{}
	desc.Html.Type = "tiptap"
	// a complete value followed by bare text without a separator reproduces the exact
	// json error seen live ("invalid character 't' after object key:value pair").
	desc.Html.Markup = `{"version":2 the rest is text}`

	item := downloadQueueItemNAPI{
		deviation: &napi.Deviation{
			DeviationId: "1204305967",
			Author:      &napi.Author{Username: "Pippers02"},
			Extended:    &napi.Extended{DescriptionText: desc},
		},
	}

	var files []string
	err := m.downloadDescriptionNapi(item, &files)

	assert.NoError(t, err, "corrupt (DeviantArt-side) description markup must not abort the deviation download")
	assert.Empty(t, files, "no description file should be written for corrupt markup")
}

// TestDownloadLiteratureNapi_CorruptMarkupErrors ensures the literature/journal body path
// stays strict: unlike the best-effort description, a corrupt primary-content body must
// surface the parse error rather than being silently skipped.
func TestDownloadLiteratureNapi_CorruptMarkupErrors(t *testing.T) {
	m := NewBareModule().ModuleInterface.(*deviantArt)

	tc := &napi.TextContent{}
	tc.Html.Type = "tiptap"
	tc.Html.Markup = `{"version":2 the rest is text}`

	item := downloadQueueItemNAPI{
		deviation: &napi.Deviation{
			DeviationId: "1204305967",
			Type:        "literature",
			Author:      &napi.Author{Username: "Pippers02"},
			TextContent: tc,
		},
	}

	var files []string
	err := m.downloadLiteratureNapi(item, &files)

	assert.Error(t, err, "corrupt literature body markup must surface an error, not be silently skipped")
	assert.Empty(t, files)
}

// TestDownloadDescriptionNapi_NilDescription ensures a deviation with an extended
// response but no description (DescriptionText == nil) is a no-op instead of a
// nil pointer dereference inside GetTextContent.
func TestDownloadDescriptionNapi_NilDescription(t *testing.T) {
	m := NewBareModule().ModuleInterface.(*deviantArt)

	item := downloadQueueItemNAPI{
		deviation: &napi.Deviation{
			DeviationId: "1204305967",
			Author:      &napi.Author{Username: "Pippers02"},
			Extended:    &napi.Extended{DescriptionText: nil},
		},
	}

	var files []string
	assert.NotPanics(t, func() {
		assert.NoError(t, m.downloadDescriptionNapi(item, &files))
	})
	assert.Empty(t, files)
}
