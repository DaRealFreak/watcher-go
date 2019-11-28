package patreon

import (
	"encoding/json"
	"fmt"
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
		URL      string `json:"url"`
		PostFile struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"post_file"`
	} `json:"attributes"`
	Relationships struct {
		AttachmentSection struct {
			Attachments []*attachmentData `json:"data"`
		} `json:"attachments"`
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
		// attributes of tiered/locked rewards
		AccessRuleType string `json:"access_rule_type"`
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

	campaignID, err := m.getCreatorCampaign(creatorID)
	if err != nil {
		return err
	}

	var postDownloads []*postDownload

	foundCurrentItem := false
	campaignPostsURI := m.getCampaignPostsURI(campaignID)
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

			postDownload := &postDownload{
				PostID: int(postID),
			}

			for _, attachment := range post.Relationships.AttachmentSection.Attachments {
				if include := m.findAttachmentInIncludes(attachment, postsData.Included); include != nil {
					postDownload.Attachments = append(postDownload.Attachments, include)
				}
			}

			postDownloads = append(postDownloads, postDownload)
		}

		// we are already on the last page
		if postsData.Links.Next == "" {
			break
		}

		campaignPostsURI = postsData.Links.Next
	}

	// reverse to add oldest posts first
	for i, j := 0, len(postDownloads)-1; i < j; i, j = i+1, j-1 {
		postDownloads[i], postDownloads[j] = postDownloads[j], postDownloads[i]
	}

	return m.processDownloadQueue(postDownloads, item)
}

// getCampaignPostsURI returns the post public API URI for the first page
func (m *patreon) getCampaignPostsURI(campaignID int) string {
	return fmt.Sprintf("https://www.patreon.com/api/posts"+
		"?include=user,attachments,user_defined_tags,campaign,poll.choices,poll.current_user_responses.user,"+
		"poll.current_user_responses.choice,poll.current_user_responses.poll,access_rules.tier.null,images.null,"+
		"audio.null"+
		"&fields[post]=change_visibility_at,comment_count,content,current_user_can_delete,current_user_can_view,"+
		"current_user_has_liked,embed,image,is_paid,like_count,min_cents_pledged_to_view,post_file,post_metadata,"+
		"published_at,patron_count,patreon_url,post_type,pledge_url,thumbnail_url,teaser_text,title,upgrade_url,url,"+
		"was_posted_by_campaign_owner"+
		"&fields[user]=image_url,full_name,url"+
		"&fields[campaign]=currency,show_audio_post_download_links,avatar_photo_url,earnings_visibility,is_nsfw,"+
		"is_monthly,name,url"+
		"&fields[access_rule]=access_rule_type,amount_cents"+
		"&fields[media]=id,image_urls,download_url,metadata,file_name"+
		"&sort=-published_at"+
		"&filter[campaign_id]=%d"+
		"&filter[is_draft]=false"+
		"&filter[contains_exclusive_posts]=true"+
		"&json-api-use-default-includes=false"+
		"&json-api-version=1.0", campaignID)
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
