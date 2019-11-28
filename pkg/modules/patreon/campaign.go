package patreon

import (
	"fmt"

	"github.com/DaRealFreak/watcher-go/pkg/models"
)

type campaignResponse struct {
	Data struct {
	} `json:"data"`
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

	fmt.Println(m.getCampaignData(campaignID))

	return nil
}

// getCampaignData returns the campaign data including the rewards and the navigation
func (m *patreon) getCampaignData(campaignID int) (string, error) {
	res, err := m.Session.Get(fmt.Sprintf("https://www.patreon.com/api/posts"+
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
		"&json-api-version=1.0", campaignID))
	if err != nil {
		return "", err
	}

	return m.Session.GetDocument(res).Text(), nil
}
