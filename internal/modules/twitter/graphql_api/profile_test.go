package graphql_api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// sampleUserByScreenNameResponse is a real (anonymous-looking) UserByScreenName
// response used to verify the profile formatting end-to-end without a network call.
const sampleUserByScreenNameResponse = `{"data":{"user":{"result":{"__typename":"User","affiliates_highlighted_label":{},"avatar":{"image_url":"https://pbs.twimg.com/profile_images/2035050567618502658/AKEiLfHv_normal.jpg"},"business_account":{},"core":{"created_at":"Sat Aug 30 23:33:22 +0000 2014","name":"クロノミツキ","screen_name":"KuroTuki_nn"},"creator_subscriptions_count":0,"dm_permissions":{"can_dm":false},"follow_request_sent":false,"grok_translated_bio_with_availability":{"is_available":false},"has_graduated_access":true,"has_hidden_subscriptions_on_profile":false,"highlights_info":{"can_highlight_tweets":true,"highlighted_tweets":"9"},"id":"VXNlcjoyNzgxMzEyMDk3","is_blue_verified":true,"is_profile_translatable":false,"legacy":{"default_profile":false,"default_profile_image":false,"description":"お絵描きします🎨\n担当：TrymenT▷https://t.co/gFIqRkieci Re:LieF▷https://t.co/OdOJz2ZHLU\nFANBOX▷https://t.co/vHm0YlILet\nFantia▷https://t.co/zjORxquSQy\n絵の動画▷https://t.co/KFKPeccakI","entities":{"description":{"urls":[{"display_url":"re-tryment.com","expanded_url":"http://re-tryment.com","indices":[20,43],"url":"https://t.co/gFIqRkieci"},{"display_url":"rask-soft.com","expanded_url":"http://rask-soft.com","indices":[52,75],"url":"https://t.co/OdOJz2ZHLU"},{"display_url":"kurono-mitsuki.fanbox.cc","expanded_url":"http://kurono-mitsuki.fanbox.cc","indices":[83,106],"url":"https://t.co/vHm0YlILet"},{"display_url":"fantia.jp/fanclubs/172663","expanded_url":"http://fantia.jp/fanclubs/172663","indices":[114,137],"url":"https://t.co/zjORxquSQy"},{"display_url":"youtube.com/@Kurono_Mitsuki","expanded_url":"http://youtube.com/@Kurono_Mitsuki","indices":[143,166],"url":"https://t.co/KFKPeccakI"}]},"url":{"urls":[{"display_url":"pixiv.net/users/12221400","expanded_url":"http://pixiv.net/users/12221400","indices":[0,23],"url":"https://t.co/myN24AYP7S"}]}},"fast_followers_count":0,"favourites_count":14868,"follow_request_sent":false,"followers_count":181404,"friends_count":296,"has_custom_timelines":true,"is_translator":false,"listed_count":1349,"media_count":765,"needs_phone_verification":false,"normal_followers_count":181404,"notifications":false,"pinned_tweet_ids_str":["2044349639416275355"],"possibly_sensitive":true,"profile_banner_url":"https://pbs.twimg.com/profile_banners/2781312097/1723678425","profile_interstitial_type":"","statuses_count":10789,"time_zone":"","translator_type":"none","url":"https://t.co/myN24AYP7S","utc_offset":0,"want_retweets":false,"withheld_description":"","withheld_scope":""},"legacy_extended_profile":{"birthdate":{"day":7,"month":8,"visibility":"Public","year_visibility":"Self"}},"location":{"location":"現在お仕事は全てお断りしております"},"media_permissions":{"can_media_tag":true},"parody_commentary_fan_label":"None","premium_gifting_eligible":false,"privacy":{"protected":false},"professional":{"category":[],"professional_type":"Creator","rest_id":"1786030719678234780"},"profile_bio":{"description":"お絵描きします🎨\n担当：TrymenT▷https://t.co/gFIqRkieci Re:LieF▷https://t.co/OdOJz2ZHLU\nFANBOX▷https://t.co/vHm0YlILet\nFantia▷https://t.co/zjORxquSQy\n絵の動画▷https://t.co/KFKPeccakI"},"profile_description_language":"ja","profile_image_shape":"Circle","profile_sort_enabled":true,"relationship_perspectives":{"blocked_by":false,"blocking":false,"followed_by":false,"following":false,"muting":false},"rest_id":"2781312097","super_follow_eligible":false,"super_followed_by":false,"super_following":false,"user_seed_tweet_count":0,"verification":{"verified":false},"verification_info":{"is_identity_verified":false,"reason":{"description":{"entities":[{"from_index":26,"ref":{"url":"https://help.twitter.com/managing-your-account/about-twitter-verified-accounts","url_type":"ExternalUrl"},"to_index":36}],"text":"This account is verified. Learn more"},"verified_since_msec":"1722949130475"}}}}}}`

func TestUser_FormatProfile(t *testing.T) {
	a := assert.New(t)

	var info UserInformation
	a.NoError(json.Unmarshal([]byte(sampleUserByScreenNameResponse), &info))

	out := info.Data.User.Result.FormatProfile()

	expected := strings.Join([]string{
		"クロノミツキ",
		"@KuroTuki_nn",
		"",
		"お絵描きします🎨",
		"担当：TrymenT▷http://re-tryment.com Re:LieF▷http://rask-soft.com",
		"FANBOX▷http://kurono-mitsuki.fanbox.cc",
		"Fantia▷http://fantia.jp/fanclubs/172663",
		"絵の動画▷http://youtube.com/@Kurono_Mitsuki",
		"",
		"Location: 現在お仕事は全てお断りしております",
		"Link: pixiv.net/users/12221400",
		"Born: August 7",
		"Joined: August 2014",
	}, "\n")

	a.Equal(expected, out)
	a.NotContains(out, "https://t.co/")
}

func TestUser_FormatProfile_Minimal(t *testing.T) {
	a := assert.New(t)

	u := User{}
	u.Core.Name = "Test"
	u.Core.ScreenName = "test_user"

	out := u.FormatProfile()
	a.Equal("Test\n@test_user", out)
}

func TestUser_FormatProfile_EmptyUser(t *testing.T) {
	a := assert.New(t)
	a.Equal("", User{}.FormatProfile())
}
