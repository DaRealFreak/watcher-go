package kemono

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/PuerkitoBio/goquery"
)

// CustomTime for parsing timestamps with fractional seconds
type CustomTime struct {
	time.Time
}

// UnmarshalJSON handles the parsing of time strings with or without fractional seconds
func (ct *CustomTime) UnmarshalJSON(b []byte) error {
	s := string(b)
	s = s[1 : len(s)-1] // Remove quotes

	// Define possible layouts
	layouts := []string{
		"2006-01-02T15:04:05.000000", // With fractional seconds
		"2006-01-02T15:04:05",        // Without fractional seconds
	}

	var err error
	for _, layout := range layouts {
		ct.Time, err = time.Parse(layout, s)
		if err == nil {
			return nil
		}
	}

	// Return the last error if no layout matched
	return fmt.Errorf("could not parse time: %s, error: %w", s, err)
}

type Root struct {
	Props               Props          `json:"props"`
	Base                Base           `json:"base"`
	Results             []Result       `json:"results"`
	ResultPreviews      [][]Thumbnail  `json:"result_previews"`
	ResultAttachments   [][]Attachment `json:"result_attachments"`
	ResultIsImage       []bool         `json:"result_is_image"`
	DisableServiceIcons bool           `json:"disable_service_icons"`
}

type Props struct {
	CurrentPage string      `json:"currentPage"`
	ID          string      `json:"id"`
	Service     string      `json:"service"`
	Name        string      `json:"name"`
	Count       int         `json:"count"`
	Limit       int         `json:"limit"`
	Artist      Artist      `json:"artist"`
	DisplayData DisplayData `json:"display_data"`
	DmCount     int         `json:"dm_count"`
	ShareCount  int         `json:"share_count"`
	HasLinks    string      `json:"has_links"`
}

type Artist struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Service    string     `json:"service"`
	Indexed    CustomTime `json:"indexed"`
	Updated    CustomTime `json:"updated"`
	PublicID   string     `json:"public_id"`
	RelationID *string    `json:"relation_id"`
}

type DisplayData struct {
	Service string `json:"service"`
	Href    string `json:"href"`
}

type Base struct {
	Service  string `json:"service"`
	ArtistID string `json:"artist_id"`
}

type Result struct {
	ID          string       `json:"id"`
	User        string       `json:"user"`
	Service     string       `json:"service"`
	Title       string       `json:"title"`
	Substring   string       `json:"substring"`
	Published   CustomTime   `json:"published"`
	File        File         `json:"file"`
	Attachments []Attachment `json:"attachments"`
}

type File struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type Attachment struct {
	Name   string  `json:"name"`
	Path   string  `json:"path"`
	Server *string `json:"server"`
}

type Thumbnail struct {
	Type   string `json:"type"`
	Server string `json:"server"`
	Name   string `json:"name"`
	Path   string `json:"path"`
}

type postItem struct {
	id    string
	title string
	uri   string
}

func (m *kemono) parseUser(item *models.TrackedItem) error {
	// update base URL
	if strings.Contains(item.URI, "coomer.su") {
		m.baseUrl, _ = url.Parse("https://coomer.su")
	} else {
		m.baseUrl, _ = url.Parse("https://kemono.su")
	}

	search := regexp.MustCompile(`https://(?:kemono|coomer).su/([^/?&]+)/user/([^/?&]+)`).FindStringSubmatch(item.URI)
	userId := ""
	service := ""
	if len(search) == 3 {
		service = search[1]
		userId = search[2]
	}

	if userId == "" || service == "" {
		return fmt.Errorf("could not extract user ID and service from URL: %s", item.URI)
	}

	apiUrl := fmt.Sprintf("%s/api/v1/%s/user/%s/posts-legacy", m.baseUrl.String(), service, userId)
	response, err := m.Session.Get(apiUrl)
	if err != nil {
		return err
	}

	out, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		return err
	}

	var root Root
	err = json.Unmarshal(out, &root)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	var (
		downloadQueue    []Result
		foundCurrentItem bool
		offset           int
	)

	for {
		// we are beyond the last page, break here
		if len(root.Results) == 0 {
			break
		}

		for _, post := range root.Results {
			// check if we reached the current item already
			if post.ID == item.CurrentItem {
				foundCurrentItem = true
				break
			}

			downloadQueue = append(downloadQueue, post)
		}

		if foundCurrentItem {
			break
		}

		// increase offset for the next page
		offset += 50
		pageUrl, _ := url.Parse(apiUrl)
		queries := pageUrl.Query()
		queries.Set("o", strconv.Itoa(offset))
		pageUrl.RawQuery = queries.Encode()

		response, _ = m.Session.Get(pageUrl.String())
		// new behavior of kemono.su is to redirect to the main page if the offset exceeds the last page
		if response.Request.URL.String() != pageUrl.String() {
			break
		}

		out, readErr = io.ReadAll(response.Body)
		if readErr != nil {
			return readErr
		}

		err = json.Unmarshal(out, &root)
		if err != nil {
			return fmt.Errorf("failed to unmarshal JSON: %w", err)
		}
	}

	// reverse download queue to download old items first
	for i, j := 0, len(downloadQueue)-1; i < j; i, j = i+1, j-1 {
		downloadQueue[i], downloadQueue[j] = downloadQueue[j], downloadQueue[i]
	}

	return m.processDownloadQueue(item, downloadQueue)
}

func (m *kemono) parsePost(item *models.TrackedItem) error {
	// update base URL
	if strings.Contains(item.URI, "coomer.su") {
		m.baseUrl, _ = url.Parse("https://coomer.su")
	} else {
		m.baseUrl, _ = url.Parse("https://kemono.su")
	}

	// extract 2nd number from example URL: https://kemono.su/patreon/user/551274/post/24446001
	postId := regexp.MustCompile(`.*([^/?&]+)/user/([^/?&]+)/post/(\d+)`).FindStringSubmatch(item.URI)
	if len(postId) != 3 {
		return fmt.Errorf("could not extract post ID from URL: %s", item.URI)
	}

	return m.processDownloadQueue(item, []Result{{
		Service: postId[1],
		User:    postId[2],
		ID:      postId[3],
	}})
}

func (m *kemono) getPostsUrls(html string) (postItems []*postItem) {
	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	document.Find("article[data-user][data-id]").Each(func(index int, row *goquery.Selection) {
		if dataId, exists := row.Attr("data-id"); exists {
			uriTag := row.Find("a[href*=\"/post/\"]")
			uri, _ := uriTag.Attr("href")
			parsedUri, _ := url.Parse(uri)
			absUri := m.baseUrl.ResolveReference(parsedUri)
			titleTag := row.Find("header.post-card__header")
			title := strings.TrimSpace(strings.ReplaceAll(titleTag.Text(), "\n", ""))

			postItems = append(postItems, &postItem{
				id:    dataId,
				title: title,
				uri:   absUri.String(),
			})
		}
	})

	return postItems
}
