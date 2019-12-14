package patreon

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/DaRealFreak/watcher-go/pkg/models"
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
	} `json:"attributes"`
	Relationships struct {
		AttachmentSection struct {
			Attachments []*attachmentData `json:"data"`
		} `json:"attachments"`
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
	ID   json.Number `json:"id"`
	Type string      `json:"type"`
}

// attachmentData is the struct of the attachment in the post data
type attachmentData struct {
	ID   json.Number `json:"id"`
	Type string      `json:"type"`
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
			postID, _ := strconv.ParseInt(post.ID.String(), 10, 64)
			if !(item.CurrentItem == "" || postID > currentItemID) {
				foundCurrentItem = true
				break
			}

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

// extractPostDownload extracts the relevant download data for from the passed post
func (m *patreon) extractPostDownload(
	creatorID int, campaign *userResponse, postID int64, post *campaignPost, postsData *campaignResponse,
) *postDownload {
	postDownload := &postDownload{
		CreatorID:   creatorID,
		CreatorName: campaign.Data.Attributes.FullName,
		PostID:      int(postID),
		PatreonURL:  post.Attributes.PatreonURL,
	}

	// ignore embedded videos
	if post.Attributes.PostType == "video_embed" {
		return nil
	}

	for _, attachment := range post.Relationships.AttachmentSection.Attachments {
		if include := m.findAttachmentInIncludes(attachment, postsData.Included); include != nil {
			postDownload.Attachments = append(postDownload.Attachments, include)
		}
	}

	for _, attachment := range post.Relationships.ImageSection.ImageAttachments {
		if include := m.findAttachmentInIncludes(attachment, postsData.Included); include != nil {
			postDownload.Attachments = append(postDownload.Attachments, include)
		}
	}

	return postDownload
}

// reverseDownloadQueue reverses the download queue to download the oldest items first
func (m *patreon) reverseDownloadQueue(downloadQueue []*postDownload) []*postDownload {
	// reverse to add oldest posts first
	for i, j := 0, len(downloadQueue)-1; i < j; i, j = i+1, j-1 {
		downloadQueue[i], downloadQueue[j] = downloadQueue[j], downloadQueue[i]
	}

	return downloadQueue
}

// getCampaignPostsURI returns the post public API URI for the first page
func (m *patreon) getCampaignPostsURI(campaignID int) string {
	values := url.Values{
		"include": {
			"user,attachments,user_defined_tags,campaign,poll.choices,poll.current_user_responses.user," +
				"poll.current_user_responses.choice,poll.current_user_responses.poll,access_rules.tier.null," +
				"images.null,audio.null",
		},
		"fields[post]": {
			"change_visibility_at,comment_count,content,current_user_can_delete,current_user_can_view," +
				"current_user_has_liked,embed,image,is_paid,like_count,min_cents_pledged_to_view,post_file," +
				"post_metadata,published_at,patron_count,patreon_url,post_type,pledge_url,thumbnail_url," +
				"teaser_text,title,upgrade_url,url,was_posted_by_campaign_owner",
		},
		"fields[user]": {"image_url,full_name,url"},
		"fields[campaign]": {
			"currency,show_audio_post_download_links,avatar_photo_url,earnings_visibility,is_nsfw,is_monthly,name,url",
		},
		"fields[access_rule]":                {"access_rule_type,amount_cents"},
		"fields[media]":                      {"id,image_urls,download_url,metadata,file_name"},
		"sort":                               {"-published_at"},
		"filter[campaign_id]":                {strconv.Itoa(campaignID)},
		"filter[exclude_inaccessible_posts]": {"true"},
		"json-api-use-default-includes":      {"false"},
		"json-api-version":                   {"1.0"},
	}

	return fmt.Sprintf("https://www.patreon.com/api/posts?%s", values.Encode())
}

// getCampaignData returns the campaign data extracted from the passed campaignPostsUri
func (m *patreon) getCampaignData(campaignPostsURI string) (*campaignResponse, error) {
	res, err := m.Session.Get(campaignPostsURI)
	if err != nil {
		return nil, err
	}

	var postsData campaignResponse
	err = json.Unmarshal([]byte(m.Session.GetDocument(res).Text()), &postsData)

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
