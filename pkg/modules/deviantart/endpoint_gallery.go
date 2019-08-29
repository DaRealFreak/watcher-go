package deviantart

import (
	"encoding/json"
	"fmt"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	"io/ioutil"
	"net/url"
	"strconv"
)

// GalleryAll implements the API endpoint https://www.deviantart.com/api/v1/oauth2/gallery/all
func (m *deviantArt) GalleryAll(username string, offset uint, limit uint) *GalleryAllResponse {
	var galleryAllResponse GalleryAllResponse
	apiURL, err := url.Parse("https://www.deviantart.com/api/v1/oauth2/gallery/all")
	raven.CheckError(err)

	// add our API values and replace the RawQuery of the apiUrl
	values := url.Values{
		"username": {username},
		"offset":   {strconv.FormatUint(uint64(offset), 10)},
		"limit":    {strconv.FormatUint(uint64(limit), 10)},
	}
	apiURL.RawQuery = values.Encode()

	res, err := m.deviantArtSession.APIGet(apiURL.String(), ScopeBrowse)
	raven.CheckError(err)

	content, err := ioutil.ReadAll(res.Body)
	raven.CheckError(err)

	// unmarshal the request content into the response struct
	raven.CheckError(json.Unmarshal(content, &galleryAllResponse))
	return &galleryAllResponse
}

// GalleryFoldersCreate implements the API endpoint https://www.deviantart.com/api/v1/oauth2/gallery/folders/create
func (m *deviantArt) GalleryFoldersCreate(folder string) *GalleryFoldersCreateResponse {
	var galleryAllResponse GalleryFoldersCreateResponse

	// add our API values and replace the RawQuery of the apiURL
	values := url.Values{
		"folder": {folder},
	}

	res, err := m.deviantArtSession.APIPost(
		"https://www.deviantart.com/api/v1/oauth2/gallery/folders/create",
		values,
		ScopeGallery+" "+ScopeBrowse,
	)
	raven.CheckError(err)

	content, err := ioutil.ReadAll(res.Body)
	raven.CheckError(err)
	fmt.Println(string(content))
	fmt.Println(res.StatusCode)

	// unmarshal the request content into the response struct
	raven.CheckError(json.Unmarshal(content, &galleryAllResponse))
	return &galleryAllResponse
}
