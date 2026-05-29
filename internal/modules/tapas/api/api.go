// Package api wraps the JSON endpoints exposed by tapas.io for module use.
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	internalHttp "github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/PuerkitoBio/goquery"
	http "github.com/bogdanfinn/fhttp"
)

const BaseURL = "https://tapas.io"

// Client is the tapas.io API client.
type Client struct {
	Session internalHttp.TlsClientSessionInterface
}

// NewClient returns a new API client wrapping the given TLS session.
func NewClient(session internalHttp.TlsClientSessionInterface) *Client {
	return &Client{Session: session}
}

// SeriesIDFromSlug resolves a series slug to its numeric series id by
// scraping the public series page HTML.
func (c *Client) SeriesIDFromSlug(slug string) (string, error) {
	pageURL := fmt.Sprintf("%s/series/%s", BaseURL, url.PathEscape(slug))
	req, err := http.NewRequest(http.MethodGet, pageURL, nil)
	if err != nil {
		return "", err
	}

	res, err := c.Session.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = res.Body.Close() }()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`data-series-id="(\d+)"`),
		regexp.MustCompile(`/series/(\d+)/(?:episodes|info|subscribe)`),
	}
	for _, pattern := range patterns {
		if m := pattern.FindStringSubmatch(string(body)); len(m) > 1 {
			return m[1], nil
		}
	}

	return "", fmt.Errorf("could not resolve series id for slug %q", slug)
}

// SeriesTitle returns the human readable series title scraped from the public
// series page.
func (c *Client) SeriesTitle(seriesID string) (string, error) {
	pageURL := fmt.Sprintf("%s/series/%s/info", BaseURL, seriesID)
	req, err := http.NewRequest(http.MethodGet, pageURL, nil)
	if err != nil {
		return "", err
	}

	res, err := c.Session.Do(req)
	if err != nil {
		return "", err
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	_ = res.Body.Close()
	if err != nil {
		return "", err
	}

	if title := strings.TrimSpace(doc.Find("meta[property='og:title']").AttrOr("content", "")); title != "" {
		return cleanSeriesTitle(title), nil
	}

	if title := strings.TrimSpace(doc.Find(".info-detail__title").First().Text()); title != "" {
		return cleanSeriesTitle(title), nil
	}

	return "", fmt.Errorf("could not extract title for series %s", seriesID)
}

// cleanSeriesTitle strips the boilerplate that tapas appends to the og:title
// of every series ("Read <title> | Tapas Web Community" and similar variants),
// leaving just the work's name.
func cleanSeriesTitle(title string) string {
	if idx := strings.Index(title, " | Tapas"); idx >= 0 {
		title = title[:idx]
	}
	title = strings.TrimSpace(strings.TrimPrefix(title, "Read "))
	return title
}

// EpisodeList fetches a single page of the series' episode list, oldest first.
func (c *Client) EpisodeList(seriesID string, page int) ([]EpisodeListItem, Pagination, error) {
	apiURL := fmt.Sprintf("%s/series/%s/episodes?page=%d&sort=OLDEST", BaseURL, seriesID, page)

	var env Envelope[EpisodeListData]
	if err := c.getJSON(apiURL, &env); err != nil {
		return nil, Pagination{}, err
	}

	items, err := parseEpisodeListBody(env.Data.Body)
	if err != nil {
		return nil, env.Data.Pagination, err
	}

	return items, env.Data.Pagination, nil
}

// Episode fetches a single episode's metadata and HTML body.
func (c *Client) Episode(episodeID string) (*EpisodeData, error) {
	apiURL := fmt.Sprintf("%s/episode/%s", BaseURL, episodeID)

	var env Envelope[EpisodeData]
	if err := c.getJSON(apiURL, &env); err != nil {
		return nil, err
	}

	return &env.Data, nil
}

// getJSON performs an XHR-flavored GET and unmarshals the response body.
func (c *Client) getJSON(apiURL string, into interface{}) error {
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")

	res, err := c.Session.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("request to %s returned status %d: %s", apiURL, res.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, into); err != nil {
		return fmt.Errorf("failed to unmarshal response from %s: %w", apiURL, err)
	}

	return nil
}

// parseEpisodeListBody extracts EpisodeListItem entries from the HTML fragment
// the episodes endpoint embeds in its JSON body field.
func parseEpisodeListBody(body string) ([]EpisodeListItem, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	var items []EpisodeListItem
	doc.Find("li[data-id]").Each(func(_ int, s *goquery.Selection) {
		id, ok := s.Attr("data-id")
		if !ok {
			return
		}
		if _, err := strconv.Atoi(id); err != nil {
			return
		}

		title := strings.TrimSpace(s.Find(".info__title").First().Text())
		if title == "" {
			title = strings.TrimSpace(s.Find(".info__label").First().Text())
		}

		items = append(items, EpisodeListItem{ID: id, Title: title})
	})

	return items, nil
}
