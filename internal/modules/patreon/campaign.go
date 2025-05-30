package patreon

import (
	"encoding/json"
	"fmt"
	http "github.com/bogdanfinn/fhttp"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/PuerkitoBio/goquery"
	log "github.com/sirupsen/logrus"
)

// campaignResponse contains the struct for most relevant data of posts
type campaignResponse struct {
	Posts    []*campaignPost    `json:"data"`
	Included []*campaignInclude `json:"included"`
	Links    struct {
		Next string `json:"next"`
	} `json:"links"`
}

// campaignPost is the struct of the posts of the campaign
type campaignPost struct {
	Attributes struct {
		URL        string `json:"url"`
		PatreonURL string `json:"patreon_url"`
		PostType   string `json:"post_type"`
		PostFile   struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"post_file"`
		Embed struct {
			Description string `json:"description"`
			Html        string `json:"html"`
			URL         string `json:"url"`
		} `json:"embed"`
		EditedAt    *Time `json:"edited_at"`
		CreatedAt   *Time `json:"created_at"`
		PublishedAt *Time `json:"published_at"`
	} `json:"attributes"`
	Relationships struct {
		AttachmentSection struct {
			Attachments []*attachmentData `json:"data"`
		} `json:"attachments"`
		AttachmentMediaSection struct {
			Attachments []*attachmentData `json:"data"`
		} `json:"attachments_media"`
		ImageSection struct {
			ImageAttachments []*attachmentData `json:"data"`
		} `json:"images"`
	} `json:"relationships"`
	ID   json.Number `json:"id"`
	Type string      `json:"type"`
}

// campaignInclude is the struct of all includes related to the posts
type campaignInclude struct {
	Attributes struct {
		// attributes of attachment types
		Name string `json:"name"`
		URL  string `json:"url"`
		// attributes of media types
		DownloadURL string `json:"download_url"`
		FileName    string `json:"file_name"`
	} `json:"attributes"`
	ID   string `json:"id"`
	Type string `json:"type"`
}

// attachmentData is the struct of the attachment in the post data
type attachmentData struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// parseCampaign is the main entry point to parse campaigns of the module
func (m *patreon) parseCampaign(item *models.TrackedItem) error {
	creatorID, err := m.getCreatorID(item.URI)
	if err != nil {
		return err
	}

	campaign, err := m.getCreatorCampaign(creatorID)
	if err != nil {
		return err
	}

	campaignID, err := strconv.ParseInt(campaign.Data.Relationships.Campaign.Data.ID.String(), 10, 64)
	if err != nil {
		return err
	}

	var postDownloads []*postDownload

	foundCurrentItem := false
	campaignPostsURI := m.getCampaignPostsURI(int(campaignID))
	currentItemID, _ := strconv.ParseInt(item.CurrentItem, 10, 64)

	for !foundCurrentItem {
		postsData, err := m.getCampaignData(campaignPostsURI)
		if err != nil {
			return err
		}

		for _, post := range postsData.Posts {
			comparisonTime := post.Attributes.EditedAt
			if comparisonTime == nil {
				comparisonTime = post.Attributes.PublishedAt
			}

			if item.CurrentItem != "" && comparisonTime.Unix() <= currentItemID {
				foundCurrentItem = true
				break
			}

			postID, _ := strconv.ParseInt(post.ID.String(), 10, 64)
			if download := m.extractPostDownload(creatorID, campaign, postID, post, postsData); download != nil {
				postDownloads = append(postDownloads, download)
			}
		}

		// we are already on the last page
		if postsData.Links.Next == "" {
			break
		}

		campaignPostsURI = postsData.Links.Next
	}

	return m.processDownloadQueue(m.reverseDownloadQueue(postDownloads), item)
}

func (m *patreon) extractIframeSources(html string) (results []string) {
	if document, err := goquery.NewDocumentFromReader(io.NopCloser(strings.NewReader(html))); err == nil {
		document.Find("iframe[src]").Each(func(i int, iframeTag *goquery.Selection) {
			srcUrl, _ := iframeTag.Attr("src")
			parsedUrl, parseErr := url.Parse(srcUrl)
			if parseErr == nil {
				switch parsedUrl.Host {
				case "cdn.embedly.com":
					if parsedUrl.Query().Has("src") {
						results = append(results, parsedUrl.Query()["src"]...)
					}
				default:
					results = append(results, srcUrl)
				}
			}
		})
	}

	return results
}

// extractPostDownload extracts the relevant download data for from the passed post
func (m *patreon) extractPostDownload(
	creatorID int, campaign *userResponse, postID int64, post *campaignPost, postsData *campaignResponse,
) *postDownload {
	comparisonTime := post.Attributes.EditedAt
	if comparisonTime == nil {
		comparisonTime = post.Attributes.PublishedAt
	}

	download := &postDownload{
		CreatorID:    creatorID,
		CreatorName:  campaign.Data.Attributes.FullName,
		PostID:       int(postID),
		PatreonURL:   post.Attributes.PatreonURL,
		ExternalURLs: []string{},
		EditedAt:     comparisonTime,
	}

	factory := modules.GetModuleFactory()
	if post.Attributes.Embed.Html != "" {
		externalUrls := m.extractIframeSources(post.Attributes.Embed.Html)
		for _, externalUrl := range externalUrls {
			if factory.CanParse(externalUrl) {
				embedUrl, _ := url.Parse(externalUrl)
				parsedQueryString, _ := url.ParseQuery(embedUrl.RawQuery)
				parsedQueryString["referer"] = []string{post.Attributes.URL}
				embedUrl.RawQuery = parsedQueryString.Encode()
				download.ExternalURLs = append(download.ExternalURLs, embedUrl.String())
			} else {
				log.WithField("module", m.Key).Warnf("unable to parse found URL: \"%s\"", externalUrl)
			}
		}
	}

	// if we have a URL without having extracted an external URL from the HTML we can add it without getting duplicates
	if len(download.ExternalURLs) == 0 && post.Attributes.Embed.URL != "" {
		if factory.CanParse(post.Attributes.Embed.URL) {
			embedUrl, _ := url.Parse(post.Attributes.Embed.URL)
			parsedQueryString, _ := url.ParseQuery(embedUrl.RawQuery)
			embedUrl.RawQuery = parsedQueryString.Encode()
			download.ExternalURLs = append(download.ExternalURLs, embedUrl.String())
		} else {
			log.WithField("module", m.Key).Warnf("unable to parse found URL: \"%s\"", post.Attributes.Embed.URL)
		}
	}

	/*
		// external URLs are still the same even if the embedded link is updated (because of f.e. wrong links)
		// caused a lot of inaccessible downloads and parsing the embedded HTML was more reliable
		if post.Attributes.Embed.URL != "" {
			if modules.GetModuleFactory().CanParse(post.Attributes.Embed.URL) {
				embedUrl, _ := url.Parse(post.Attributes.Embed.URL)
				parsedQueryString, _ := url.ParseQuery(embedUrl.RawQuery)
				parsedQueryString["referer"] = []string{post.Attributes.PatreonURL}
				embedUrl.RawQuery = parsedQueryString.Encode()
				download.ExternalURLs = append(download.ExternalURLs, embedUrl.String())
			} else {
				log.Warnf("unable to parse found URL: \"%s\"", post.Attributes.Embed.URL)
			}
		}
	*/

	// older posts have attachments still in the attachments section (newer posts in the attachments_media)
	for _, attachment := range post.Relationships.AttachmentSection.Attachments {
		if include := m.findAttachmentInIncludes(attachment, postsData.Included); include != nil {
			download.Attachments = append(download.Attachments, include)
		}
	}

	// patreon moved newer posts to the attachment media section
	for _, attachment := range post.Relationships.AttachmentMediaSection.Attachments {
		if include := m.findAttachmentInIncludes(attachment, postsData.Included); include != nil {
			download.Attachments = append(download.Attachments, include)
		}
	}

	// ignore embedded videos and links, since the chance for actual images instead of link previews is low
	if post.Attributes.PostType != "video_embed" && post.Attributes.PostType != "link" {
		for _, attachment := range post.Relationships.ImageSection.ImageAttachments {
			if include := m.findAttachmentInIncludes(attachment, postsData.Included); include != nil {
				download.Attachments = append(download.Attachments, include)
			}
		}
	}

	return download
}

// reverseDownloadQueue reverses the download queue to download the oldest items first
func (m *patreon) reverseDownloadQueue(downloadQueue []*postDownload) []*postDownload {
	// reverse to add the oldest posts first
	for i, j := 0, len(downloadQueue)-1; i < j; i, j = i+1, j-1 {
		downloadQueue[i], downloadQueue[j] = downloadQueue[j], downloadQueue[i]
	}

	return downloadQueue
}

// getCampaignPostsURI returns the post public API URI for the first page
func (m *patreon) getCampaignPostsURI(campaignID int) string {
	values := url.Values{
		"fields[access_rule]": {
			"access_rule_type,amount_cents",
		},
		"fields[campaign]": {
			"currency,show_audio_post_download_links,avatar_photo_url,avatar_photo_image_urls,earnings_visibility," +
				"is_nsfw,is_monthly,name,url",
		},
		"fields[media]": {
			"id,image_urls,display,download_url,metadata,file_name",
		},
		"fields[post]": {
			"change_visibility_at,comment_count,commenter_count,content,created_at,current_user_can_comment," +
				"current_user_can_delete,current_user_can_report,current_user_can_view," +
				"current_user_comment_disallowed_reason,current_user_has_liked,embed,image,insights_last_updated_at," +
				"is_paid,like_count,meta_image_url,min_cents_pledged_to_view,monetization_ineligibility_reason," +
				"post_file,post_metadata,published_at,patreon_url,post_type,pledge_url,preview_asset_type,thumbnail," +
				"thumbnail_url,teaser_text,title,upgrade_url,url,was_posted_by_campaign_owner,has_ti_violation," +
				"moderation_status,post_level_suspension_removal_date,pls_one_liners_by_category,video,video_preview," +
				"view_count,content_unlock_options,is_new_to_current_user,watch_state," +
				// custom field not used in the normal web request but documented in the API
				"edited_at",
		},
		"fields[user]":                       {"image_url,full_name,url"},
		"filter[campaign_id]":                {strconv.Itoa(campaignID)},
		"filter[exclude_inaccessible_posts]": {"true"},
		"include": {
			"campaign,access_rules,access_rules.tier.null,attachments_media,audio,audio_preview.null,drop,images," +
				"media,native_video_insights,poll.choices,poll.current_user_responses.user," +
				"poll.current_user_responses.choice,poll.current_user_responses.poll,user," +
				"user_defined_tags,ti_checks,video.null,content_unlock_options.product_variant.null",
		},
		"json-api-use-default-includes": {"false"},
		"json-api-version":              {"1.0"},
		// created_at, published_at, edited_at seem to be valid options
		"sort": {"-edited_at"},
	}

	return fmt.Sprintf("https://www.patreon.com/api/posts?%s", values.Encode())
}

// getCampaignData returns the campaign data extracted from the passed campaignPostsUri
func (m *patreon) getCampaignData(campaignPostsURI string) (*campaignResponse, error) {
	client := m.Session.GetClient()
	req, _ := http.NewRequest("GET", campaignPostsURI, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:108.0) Gecko/20100101 Firefox/108.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	readerRes, readerErr := io.ReadAll(res.Body)

	raven.CheckError(readerErr)

	var postsData campaignResponse
	err = json.Unmarshal(readerRes, &postsData)

	return &postsData, err
}

// findAttachmentInIncludes looks for an included attachments by the attachment ID
func (m *patreon) findAttachmentInIncludes(attachment *attachmentData, includes []*campaignInclude) *campaignInclude {
	for _, include := range includes {
		if include.ID == attachment.ID {
			return include
		}
	}

	return nil
}
