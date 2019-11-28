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

type campaignPost struct {
	Attributes struct {
		URL      string `json:"url"`
		PostFile struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"post_file"`
	} `json:"attributes"`
	Relationships struct {
		Attachments struct {
		} `json:"attachments"`
	} `json:"relationships"`
	ID   json.Number `json:"id"`
	Type string      `json:"type"`
}

type campaignInclude struct {
	Attributes struct {
		// attributes of attachment types
		Name string `json:"name"`
		URL  string `json:"url"`
		// attributes of media types
		DownloadURL string `json:"download_url"`
		FileName    string `json:"file_name"`
		// attributes of tiered/locked rewards
		AccessRuleType string `json:"access_rule_type"`
	} `json:"attributes"`
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

	var campaignIncludes []*campaignInclude

	foundCurrentItem := false
	campaignPostsURI := m.getCampaignPostsURI(campaignID)
	currentItemID, _ := strconv.ParseInt(item.CurrentItem, 10, 64)

	for !foundCurrentItem {
		postsData, err := m.getCampaignData(campaignPostsURI)
		if err != nil {
			return err
		}

		for _, post := range postsData.Posts {
			fmt.Println(post.ID)
		}

		// we are already on the last page
		if postsData.Links.Next == "" {
			break
		}

		campaignPostsURI = postsData.Links.Next
	}

	fmt.Println(campaignIncludes, currentItemID)

	return nil
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
