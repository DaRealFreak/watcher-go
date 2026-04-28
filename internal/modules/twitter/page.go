package twitter

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/twitter/graphql_api"
	"log/slog"
)

func (m *twitter) parsePageGraphQLApi(item *models.TrackedItem, screenName string) (err error) {
	if m.settings.UseSubFolderForAuthorName && item.SubFolder == "" {
		m.DbIO.ChangeTrackedItemSubFolder(item, screenName)
	}

	isNormalizedURI := m.normalizedUriRegexp.MatchString(item.URI)

	var userId string
	var userInformation *graphql_api.UserInformation

	if isNormalizedURI {
		userId, err = m.extractId(item.URI)
		if err != nil {
			return err
		}

		// Profile fetch is deferred until after timeline parsing: twitter does NOT
		// redirect old screen names after a rename, so calling UserByUsername with
		// the (potentially stale) screen name from the URI would miss the renamed
		// user. We still know userId from the URI, so the timeline call works
		// regardless, and we read the current screen name back from the tweets.
	} else {
		var userErr error
		userInformation, userErr = m.twitterGraphQlAPI.UserByUsername(screenName)
		if userErr != nil {
			return userErr
		}

		userId = userInformation.Data.User.Result.RestID.String()

		if applyErr := m.applyProfileResponse(item, userId, screenName, userInformation.Data.User.Result); applyErr != nil {
			return applyErr
		}
	}

	currentItemID, _ := strconv.ParseInt(item.CurrentItem, 10, 64)

	var (
		foundCurrentItem bool

		// user media variables
		timeline       graphql_api.TimelineInterface
		timelineErr    error
		bottomCursor   string
		newMediaTweets []*graphql_api.Tweet

		// search variables
		searchTime          *time.Time
		searchedMediaTweets []*graphql_api.Tweet
	)

	for !foundCurrentItem {
		if searchTime != nil && len(newMediaTweets) > 0 {
			timeline, timelineErr = m.twitterGraphQlAPI.Search(
				screenName,
				*searchTime,
				bottomCursor,
			)
		} else {
			timeline, timelineErr = m.twitterGraphQlAPI.UserTimelineV2(
				userId,
				bottomCursor,
			)
		}
		if timelineErr != nil {
			return timelineErr
		}

		if nilCount := timeline.NilItemCount(); nilCount > 0 {
			slog.Warn(fmt.Sprintf("encountered %d nil tweet items in timeline response for user %s (uri: %s, cursor: %s)",
				nilCount, screenName, item.URI, bottomCursor), "module", m.ModuleKey())
		}

		tombstoneEntries := timeline.TombstoneEntries()
		if len(tombstoneEntries) > 0 {
			slog.Warn(fmt.Sprintf("found %d tombstone entries for user %s, check your profile for location settings and IP location, skipping",
				len(tombstoneEntries),
				screenName), "module", m.ModuleKey())
			return nil
		}

		tweetEntries := timeline.TweetEntries(userId)
		if len(tweetEntries) == 0 && searchTime == nil && bottomCursor == "" {
			user := userInformation
			if user == nil {
				var userErr error
				user, userErr = m.twitterGraphQlAPI.UserByUsername(screenName)
				if userErr != nil {
					return userErr
				}
			}

			if applyErr := m.applyProfileResponse(item, userId, screenName, user.Data.User.Result); applyErr != nil {
				return applyErr
			}

			if user.Data.User.Result.Privacy.Protected {
				followRequestSent := user.Data.User.Result.Legacy.FollowRequestSent
				if followRequestSent == nil || !*followRequestSent {
					slog.Warn(fmt.Sprintf("user %s is protected, but no follow request sent, skipping",
						screenName), "module", m.ModuleKey())
				} else {
					slog.Warn(fmt.Sprintf("user %s is protected, but follow request sent, waiting for approval",
						screenName), "module", m.ModuleKey())
				}
			} else if user.Data.User.Result.TypeName != nil && *user.Data.User.Result.TypeName == "UserUnavailable" {
				// we only want to check entries if we are not in search mode, and we should have at least one entry
				slog.Warn(fmt.Sprintf("user %s is unavailable anymore, reason from twitter: %s",
					screenName,
					*user.Data.User.Result.Reason), "module", m.ModuleKey())
			} else {
				// we only want to check entries if we are not in search mode, and we should have at least one entry
				slog.Warn(fmt.Sprintf("no tweet entries found for user %s, possibly deleted",
					screenName), "module", m.ModuleKey())
			}

			return nil
		}

		for _, tweet := range tweetEntries {
			if m.settings.ConvertNameToId &&
				tweet.Item.ItemContent.TweetResults.Result.TweetData().Core.UserResults.Result.Core.ScreenName != screenName {
				screenName = tweet.Item.ItemContent.TweetResults.Result.TweetData().Core.UserResults.Result.Core.ScreenName
				if screenName != "" {
					uri := fmt.Sprintf(
						"twitter:graphQL/%s/%s",
						userId,
						tweet.Item.ItemContent.TweetResults.Result.TweetData().Core.UserResults.Result.Core.ScreenName,
					)

					slog.Warn(fmt.Sprintf("author changed its name, updated tracked uri from \"%s\" to \"%s\"",
						item.URI,
						uri), "module", m.ModuleKey())

					m.DbIO.ChangeTrackedItemUri(item, uri)
				}
			}

			itemID, _ := strconv.ParseInt(tweet.Item.ItemContent.TweetResults.Result.TweetData().RestID.String(), 10, 64)
			if itemID <= currentItemID {
				foundCurrentItem = true
				break
			}

			newMediaTweets = append(newMediaTweets, tweet)
			if searchTime != nil {
				// append to additional tweets if we are in search mode to retrieve new search time later on
				searchedMediaTweets = append(searchedMediaTweets, tweet)
			}
		}

		bottomCursor = timeline.BottomCursor()

		if searchTime != nil && (len(tweetEntries) == 0 || bottomCursor == "") {
			if len(searchedMediaTweets) > 0 {
				// search is nearly random, so try to bring it in chronological order at least for the current items
				sort.SliceStable(searchedMediaTweets, func(i, j int) bool {
					return searchedMediaTweets[i].Item.ItemContent.TweetResults.Result.TweetData().Legacy.CreatedAt.Unix() < searchedMediaTweets[j].Item.ItemContent.TweetResults.Result.TweetData().Legacy.CreatedAt.Unix()
				})

				tmp := searchedMediaTweets[0].Item.ItemContent.TweetResults.Result.TweetData().Legacy.CreatedAt.AddDate(0, 0, 1)
				// new search date is same as the previous one, break here
				if tmp.Unix() == searchTime.Unix() {
					break
				}

				// search until the previous day of the last post (in case of multiple posts on the same day)
				searchTime = &tmp

				// reset searched media tweets
				searchedMediaTweets = []*graphql_api.Tweet{}
				bottomCursor = ""
			} else {
				break
			}
		}

		if len(tweetEntries) == 0 {
			if searchTime == nil && len(newMediaTweets) > 0 {
				// search until previous day of the last post (in case of multiple posts on the same day) and reset cursor
				tmp := newMediaTweets[len(newMediaTweets)-1].Item.ItemContent.TweetResults.Result.TweetData().Legacy.CreatedAt.AddDate(0, 0, 1)
				searchTime = &tmp
				bottomCursor = ""
			} else if len(newMediaTweets) == 0 {
				// no later page cursor available and no new tweets found to retrieve search date from, so break here
				break
			}
		}
	}

	// Profile capture / follow check for normalized URI items. Done now (rather than
	// up front) because the rename detection inside the timeline loop above may have
	// updated screenName to the user's current handle — calling UserByUsername with
	// the stale name from the URI before then would miss renamed accounts.
	if isNormalizedURI {
		finalUser, finalErr := m.twitterGraphQlAPI.UserByUsername(screenName)
		if finalErr != nil {
			slog.Warn(fmt.Sprintf("unable to fetch user info for %s: %s",
				screenName, finalErr.Error()), "module", m.ModuleKey())
		} else {
			if applyErr := m.applyProfileResponse(item, userId, screenName, finalUser.Data.User.Result); applyErr != nil {
				return applyErr
			}
		}
	}

	for i, j := 0, len(newMediaTweets)-1; i < j; i, j = i+1, j-1 {
		newMediaTweets[i], newMediaTweets[j] = newMediaTweets[j], newMediaTweets[i]
	}

	if downloadErr := m.processDownloadQueueGraphQL(newMediaTweets, item); downloadErr != nil {
		return downloadErr
	}

	return nil
}

// applyProfileResponse persists the resolved profile to generated_notes and applies
// the auto-follow rules for a single UserByUsername response. Centralized so both
// the up-front fetch (non-normalized URI), the post-timeline fetch (normalized URI),
// and the no-entries fallback share identical behavior.
func (m *twitter) applyProfileResponse(item *models.TrackedItem, userId string, screenName string, user graphql_api.User) error {
	m.persistGeneratedNotes(item, userId, user)

	if !user.RelationshipPerspectives.Following && (user.Legacy.FollowRequestSent == nil || !*user.Legacy.FollowRequestSent) {
		if m.settings.FollowUser || (m.settings.FollowFavorites && item.Favorite) {
			if followErr := m.twitterGraphQlAPI.FollowUser(userId); followErr != nil {
				return followErr
			}

			slog.Info(fmt.Sprintf("followed user %s", screenName), "module", m.ModuleKey())
		}
	}

	return nil
}

func (m *twitter) parsePage(item *models.TrackedItem) error {
	screenName, screenNameErr := m.extractScreenName(item.URI)
	if screenNameErr != nil {
		return screenNameErr
	}

	return m.parsePageGraphQLApi(item, screenName)
}

func (m *twitter) extractId(uri string) (string, error) {
	if m.normalizedUriRegexp.MatchString(uri) {
		results := regexp.MustCompile(`twitter:(?:graphQL|api)/(\d+)/.*`).FindStringSubmatch(uri)
		if len(results) != 2 {
			return "", fmt.Errorf("unexpected amount of results during screen name extraction of uri %s", uri)
		}

		return results[1], nil
	}

	return "", fmt.Errorf("uri \"%s\" not matching the regular expression", uri)
}

// persistGeneratedNotes renders the resolved twitter user as a profile-view-style
// snapshot and stores it in the generated_notes column of the tracked item. Used to
// keep a local snapshot of the profile around so we still have something readable
// after the upstream profile is deleted or suspended.
//
// The expectedUserId guard protects against handle reuse: twitter does not redirect
// renamed screen names, so a stale screen name we still hold in a URI may now be
// owned by an unrelated user. Saving that user's profile to OUR tracked item would
// be wrong, so we skip when the response's rest_id does not match.
//
// Also skipped when the response does not contain a real user (UserUnavailable, or
// rest_id missing entirely) to avoid clobbering previously captured data.
func (m *twitter) persistGeneratedNotes(item *models.TrackedItem, expectedUserId string, user graphql_api.User) {
	if user.RestID.String() == "" {
		return
	}
	if expectedUserId != "" && user.RestID.String() != expectedUserId {
		return
	}
	if user.TypeName != nil && *user.TypeName == "UserUnavailable" {
		return
	}

	notes := user.FormatProfile()
	if notes == "" || notes == item.GeneratedNotes {
		return
	}

	m.DbIO.UpdateTrackedItemGeneratedNotes(item, notes)
}

func (m *twitter) extractScreenName(uri string) (string, error) {
	var results []string
	if m.normalizedUriRegexp.MatchString(uri) {
		results = regexp.MustCompile(`twitter:(?:graphQL|api)/\d+/(.*)`).FindStringSubmatch(uri)
	} else {
		results = regexp.MustCompile(`.*x.com/([^/]+)?(?:$|/)`).FindStringSubmatch(uri)
	}

	if len(results) != 2 {
		return "", fmt.Errorf("unexpected amount of results during screen name extraction of uri %s", uri)
	}

	return results[1], nil
}
