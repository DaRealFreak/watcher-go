package pixiv

import (
	"encoding/json"
	"io/ioutil"
	"net/url"
	"strconv"
	"strings"

	"github.com/DaRealFreak/watcher-go/pkg/models"
)

// parseSearch parses search words using the previous API (no search limitations)
func (m *pixiv) parseSearch(item *models.TrackedItem) (err error) {
	searchWord, err := m.getSearchWordFromURL(item.URI)
	if err != nil {
		return err
	}

	var downloadQueue []*downloadQueueItem

	foundCurrentItem := false
	page := 1

	for !foundCurrentItem {
		response, err := m.getPublicSearch(
			searchWord,
			m.getPublicSearchTargetFromURL(item.URI),
			m.getPublicSearchTypesFromURL(item.URI),
			page,
		)
		if err != nil {
			return err
		}

		for _, publicIllustration := range response.Illustrations {
			illustrationResponse, err := m.getIllustrationFromPublicIllustration(publicIllustration)
			if err != nil {
				return err
			}

			// will return 0 on error, so fine for us too
			currentItemID, _ := strconv.ParseInt(item.CurrentItem, 10, 64)
			itemID, _ := strconv.ParseInt(illustrationResponse.Illustration.ID.String(), 10, 64)
			if item.CurrentItem == "" || itemID > currentItemID {
				err = m.parseWork(illustrationResponse.Illustration, &downloadQueue)
				if err != nil {
					return err
				}
			} else {
				foundCurrentItem = true
				break
			}
		}

		if response.Pagination.Next != "" {
			if page64, err := response.Pagination.Next.Int64(); err == nil {
				page = int(page64)
			}
		} else {
			// break if we don't have another page
			break
		}
	}

	// reverse download queue to download old items first
	for i, j := 0, len(downloadQueue)-1; i < j; i, j = i+1, j-1 {
		downloadQueue[i], downloadQueue[j] = downloadQueue[j], downloadQueue[i]
	}

	// update download tag to our search word
	for _, queueItem := range downloadQueue {
		queueItem.DownloadTag = searchWord
	}

	return m.processDownloadQueue(downloadQueue, item)
}

// getPublicSearch uses the previous API to retrieve search results for the passed search word
func (m *pixiv) getPublicSearch(word string, searchMode string, searchTypes []string, page int) (
	apiRes *publicSearchResponse, err error,
) {
	var userWorks publicSearchResponse

	apiURL, _ := url.Parse("https://public-api.secure.pixiv.net/v1/search/works.json")
	data := url.Values{
		"q":                    {word},
		"page":                 {strconv.Itoa(page)},
		"per_page":             {"1000"},
		"period":               {"all"},
		"order":                {"desc"},
		"sort":                 {"date"},
		"mode":                 {searchMode},
		"types":                {strings.Join(searchTypes, ",")},
		"include_stats":        {"true"},
		"include_sanity_level": {"true"},
		"image_sizes":          {"px_128x128,px_480mw,large"},
	}
	apiURL.RawQuery = data.Encode()

	res, err := m.pixivSession.PublicAPI.Get(apiURL.String())
	if err != nil {
		return nil, err
	}

	response, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(response, &userWorks)

	return &userWorks, err
}

// getIllustrationFromPublicIllustration converts an old API illustration response into an illustration response
// like in the current newer API. As a result we can use the previous search API without any limitations.
func (m *pixiv) getIllustrationFromPublicIllustration(publicIllustration *publicIllustration) (
	apiRes *illustrationDetailResponse, err error,
) {
	// manga types don't return all pages in the search result, so we have to request the pages for each result
	if publicIllustration.Type == PublicAPISearchFilterManga {
		return m.getIllustDetail(publicIllustration.ID.String())
	}

	illustration := &illustration{
		ID:             publicIllustration.ID,
		Title:          publicIllustration.Title,
		Type:           publicIllustration.Type,
		ImageUrls:      publicIllustration.ImageUrls,
		Caption:        publicIllustration.Caption,
		Restrict:       publicIllustration.Restrict,
		User:           publicIllustration.User,
		Tags:           []*tag{},
		Tools:          publicIllustration.Tools,
		CreateDate:     publicIllustration.CreateDate,
		PageCount:      publicIllustration.PageCount,
		Width:          publicIllustration.Width,
		Height:         publicIllustration.Height,
		SanityLevel:    publicIllustration.SanityLevel,
		XRestrict:      publicIllustration.XRestrict,
		Series:         publicIllustration.Series,
		MetaSinglePage: publicIllustration.MetaSinglePage,
		MetaPages:      publicIllustration.MetaPages,
		TotalView:      publicIllustration.TotalView,
		TotalBookmarks: publicIllustration.TotalBookmarks,
		IsBookmarked:   publicIllustration.IsBookmarked,
		Visible:        publicIllustration.Visible,
		IsMuted:        publicIllustration.IsMuted,
		TotalComments:  publicIllustration.TotalComments,
	}

	if publicIllustration.Type == PublicAPISearchFilterIllustration {
		// illustration got changed to illust
		illustration.Type = SearchFilterIllustration
		// ImageUrls -> large got changed to MetaSinglePage -> original_image_url
		illustration.MetaSinglePage = map[string]string{
			"original_image_url": publicIllustration.ImageUrls["large"],
		}
	}

	// tags now got translations
	for _, tagName := range publicIllustration.Tags {
		illustration.Tags = append(illustration.Tags, &tag{
			Name: tagName,
		})
	}

	return &illustrationDetailResponse{
		Illustration: illustration,
	}, nil
}

// getPublicSearchTargetFromURL returns the search mode based from the passed URI
// default search mode is the partial tag match
func (m *pixiv) getPublicSearchTargetFromURL(uri string) string {
	u, _ := url.Parse(uri)
	q, _ := url.ParseQuery(u.RawQuery)
	// no search mode or partial tag search mod got defined
	if len(q["s_mode"]) == 0 || q["s_mode"][0] == "s_tag" {
		return PublicAPISearchModePartialTagMatch
	}
	// full tag search mode got defined
	if q["s_mode"][0] == "s_tag_full" {
		return PublicAPISearchModeExactTagMatch
	}
	// current web doesn't differentiate between tag and description anymore, so we use text as default
	return PublicAPISearchModeText
}

// getPublicSearchTypesFromURL returns the search type based from the passed URI
// default search types are all types
func (m *pixiv) getPublicSearchTypesFromURL(uri string) []string {
	u, _ := url.Parse(uri)
	q, _ := url.ParseQuery(u.RawQuery)

	// if type specified return only the specified type
	if len(q["type"]) > 0 {
		switch q["type"][0] {
		case "ugoira":
			return []string{"ugoira"}
		case "illust":
			return []string{"illustration"}
		case "manga":
			return []string{"manga"}
		}
	}
	// else return all types
	return []string{"illustration", "manga", "ugoira"}
}
