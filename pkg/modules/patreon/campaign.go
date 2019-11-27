package patreon

import (
	"fmt"
	"github.com/DaRealFreak/watcher-go/pkg/models"
)

// parseCampaign is the main entry point to parse campaigns of the module
func (m *patreon) parseCampaign(item *models.TrackedItem) error {
	creatorID, err := m.getCreatorID(item.URI)
	if err != nil {
		return err
	}

	campaignID, err := m.getCampaign(creatorID)
	if err != nil {
		return err
	}

	fmt.Println(m.getCampaignData(campaignID))
	return nil
}

// getCampaign returns the campaign ID of the creator ID
func (m *patreon) getCampaign(creatorID int) (int, error) {
	res, err := m.Session.Get(fmt.Sprintf("https://www.patreon.com/api/user/%d", creatorID))
	if err != nil {
		return 0, err
	}

	fmt.Println(m.Session.GetDocument(res).Text())
	return 0, nil
}

// getCampaignData returns the campaign data including the rewards and the navigation
func (m *patreon) getCampaignData(campaignID int) (string, error) {
	res, err := m.Session.Get(fmt.Sprintf("https://www.patreon.com/api/campaigns/%d"+
		"?include=creator.pledges.campaign.null,rss_auth_token,access_rules.tier.null"+
		"&fields[pledge]=amount_cents,created_at"+
		"&fields[campaign]=avatar_photo_url,creation_name,has_community,is_nsfw,name,url"+
		"&fields[access_rule]=access_rule_type,ammount_cents,post_count"+
		"&json-api-version=1.0", campaignID))
	if err != nil {
		return "", err
	}

	return m.Session.GetDocument(res).Text(), nil
}
