package api

import (
	"fmt"
	"net/url"
)

// TagAndWikiResponse is the partial response of the tag-and-wiki endpoint. Only the
// fields required for alias resolution are modelled.
type TagAndWikiResponse struct {
	Tag tagAndWikiTag `json:"tag"`
}

type tagAndWikiTag struct {
	Name      string `json:"name"`
	TagName   string `json:"tagName"`
	PostCount int    `json:"post_count"`
	// AliasOf is a value (not a pointer) on purpose: the API returns an empty object
	// "{}" - not null and not an omitted field - for a canonical tag.
	AliasOf AliasTag `json:"alias_of"`
}

// AliasTag is the minimal shape of an aliased-to tag reference.
type AliasTag struct {
	TagName string `json:"tagName"`
	Name    string `json:"name"`
}

// AliasTarget returns the canonical tag this response is an alias of, and whether it
// is aliased at all. A canonical tag (alias_of == {}) returns ("", false).
func (r *TagAndWikiResponse) AliasTarget() (canonical string, aliased bool) {
	// the canonical tag is the machine slug alias_of.tagName (e.g. "centorea_shianus"),
	// never the display alias_of.name ("Centorea Shianus"). Compare against the
	// requested tag's own slug tagName, and treat a self-alias (tag aliased to itself)
	// as not aliased to avoid a no-op rewrite.
	target := r.Tag.AliasOf.TagName
	if target == "" || target == r.Tag.TagName {
		return "", false
	}

	return target, true
}

// GetTagAndWiki retrieves the tag-and-wiki record for a single tag name.
func (a *SankakuComplexApi) GetTagAndWiki(tag string) (*TagAndWikiResponse, error) {
	apiURI := fmt.Sprintf(
		"https://sankakuapi.com/tag-and-wiki/name/%s?lang=en",
		url.QueryEscape(tag),
	)

	response, responseErr := a.get(apiURI)
	if responseErr != nil {
		return nil, responseErr
	}

	var apiRes TagAndWikiResponse
	if err := a.parseAPIResponse(response, &apiRes); err != nil {
		return nil, err
	}

	return &apiRes, nil
}
