package twitter

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/twitter/api"
	"github.com/DaRealFreak/watcher-go/internal/modules/twitter/graphql_api"
	log "github.com/sirupsen/logrus"
)

func (m *twitter) parsePageGraphQLApi(item *models.TrackedItem, screenName string) (err error) {
	if m.settings.UseSubFolderForAuthorName && item.SubFolder == "" {
		m.DbIO.ChangeTrackedItemSubFolder(item, screenName)
	}

	var userId string
	if m.normalizedUriRegexp.MatchString(item.URI) {
		userId, err = m.extractId(item.URI)
		if err != nil {
			return err
		}
	} else {
		userInformation, userErr := m.twitterGraphQlAPI.UserByUsername(screenName)
		if userErr != nil {
			return userErr
		}

		userId = userInformation.Data.User.Result.RestID.String()
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

		tweetEntries := timeline.TweetEntries(userId)
		for _, tweet := range tweetEntries {
			if m.settings.ConvertNameToId &&
				tweet.Content.ItemContent.TweetResults.Result.Core.UserResults.Result.Legacy.ScreenName != screenName {
				screenName = tweet.Content.ItemContent.TweetResults.Result.Core.UserResults.Result.Legacy.ScreenName
				if screenName != "" {
					uri := fmt.Sprintf(
						"twitter:graphQL/%s/%s",
						userId,
						tweet.Content.ItemContent.TweetResults.Result.Core.UserResults.Result.Legacy.ScreenName,
					)

					log.WithField("module", m.ModuleKey()).Warnf(
						"author changed its name, updated tracked uri from \"%s\" to \"%s\"",
						item.URI,
						uri,
					)

					m.DbIO.ChangeTrackedItemUri(item, uri)
				}
			}

			itemID, _ := strconv.ParseInt(tweet.Content.ItemContent.TweetResults.Result.RestID.String(), 10, 64)
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

		if searchTime != nil && len(tweetEntries) == 0 {
			if len(searchedMediaTweets) > 0 {
				// search is nearly random, so try to bring it in chronological order at least for the current items
				sort.SliceStable(searchedMediaTweets, func(i, j int) bool {
					return searchedMediaTweets[i].Content.ItemContent.TweetResults.Result.Legacy.CreatedAt.Time.Unix() < searchedMediaTweets[j].Content.ItemContent.TweetResults.Result.Legacy.CreatedAt.Time.Unix()
				})

				tmp := searchedMediaTweets[0].Content.ItemContent.TweetResults.Result.Legacy.CreatedAt.Time.AddDate(0, 0, 1)
				// new search date is same as the previous one, break here
				if tmp.Unix() == searchTime.Unix() {
					break
				}

				// search until previous day of the last post (in case of multiple posts on the same day)
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
				tmp := newMediaTweets[len(newMediaTweets)-1].Content.ItemContent.TweetResults.Result.Legacy.CreatedAt.Time.AddDate(0, 0, 1)
				searchTime = &tmp
				bottomCursor = ""
			} else if len(newMediaTweets) == 0 {
				// no later page cursor available and no new tweets found to retrieve search date from, so break here
				break
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

func (m *twitter) parsePageDeveloperApi(item *models.TrackedItem, screenName string) (err error) {
	var userId string
	var userName string
	if m.normalizedUriRegexp.MatchString(item.URI) {
		userId, err = m.extractId(item.URI)
		if err != nil {
			return err
		}

		userName, _ = m.extractScreenName(item.URI)
	} else {
		userInformation, userErr := m.twitterAPI.UserByUsername(screenName)
		if userErr != nil {
			return userErr
		}

		userId = userInformation.Data.ID.String()
		userName = userInformation.Data.Username
	}

	if m.settings.UseSubFolderForAuthorName && item.SubFolder == "" {
		m.DbIO.ChangeTrackedItemSubFolder(item, userName)
	}

	var (
		paginationToken string
		latestTweetID   string
		newMediaTweets  []api.TweetV2
	)

	for {
		tweets, timeLineErr := m.twitterAPI.UserTimelineV2(userId, item.CurrentItem, "", paginationToken)
		if timeLineErr != nil {
			return timeLineErr
		}

		if latestTweetID == "" && len(tweets.Data) > 0 {
			latestTweetID = tweets.Data[0].ID.String()
		}

		// parse all media tweets we can find
		for _, tweet := range tweets.Data {
			if strings.HasPrefix(tweet.Text, "RT @") {
				continue
			}

			if m.settings.ConvertNameToId && tweet.AuthorName != screenName {
				screenName = tweet.AuthorName
				if screenName != "" {
					uri := fmt.Sprintf(
						"twitter:api/%s/%s",
						userId,
						tweet.AuthorName,
					)

					log.WithField("module", m.ModuleKey()).Warnf(
						"author changed its name, updated tracked uri from \"%s\" to \"%s\"",
						item.URI,
						uri,
					)

					m.DbIO.ChangeTrackedItemUri(item, uri)
				}
			}

			if len(tweet.Attachments.MediaKeys) > 0 {
				for _, mediaKey := range tweet.Attachments.MediaKeys {
					for _, media := range tweets.Includes.Media {
						if media.MediaKey == mediaKey {
							tweet.Attachments.Media = append(tweet.Attachments.Media, media)
							break
						}
					}
				}

				// attach username to the tweet
				tweet.AuthorName = userName

				newMediaTweets = append(newMediaTweets, tweet)
			}
		}

		if len(tweets.Data) == 0 || tweets.Meta.NextToken == "" {
			break
		}

		// set token to check the next page
		paginationToken = tweets.Meta.NextToken
	}

	for i, j := 0, len(newMediaTweets)-1; i < j; i, j = i+1, j-1 {
		newMediaTweets[i], newMediaTweets[j] = newMediaTweets[j], newMediaTweets[i]
	}

	if downloadErr := m.processDownloadQueueDeveloperApi(newMediaTweets, item); downloadErr != nil {
		return downloadErr
	}

	if latestTweetID != "" {
		// update also if no new media item got found to possibly save some queries if media tweets happen later
		m.DbIO.UpdateTrackedItem(item, latestTweetID)
	}

	return nil
}

func (m *twitter) parsePage(item *models.TrackedItem) error {
	screenName, screenNameErr := m.extractScreenName(item.URI)
	if screenNameErr != nil {
		return screenNameErr
	}

	if m.settings.Api.UseGraphQlApi {
		return m.parsePageGraphQLApi(item, screenName)
	} else {
		return m.parsePageDeveloperApi(item, screenName)
	}
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

func (m *twitter) extractScreenName(uri string) (string, error) {
	var results []string
	if m.normalizedUriRegexp.MatchString(uri) {
		results = regexp.MustCompile(`twitter:(?:graphQL|api)/\d+/(.*)`).FindStringSubmatch(uri)
	} else {
		results = regexp.MustCompile(`.*twitter.com/([^/]+)?(?:$|/)`).FindStringSubmatch(uri)
	}

	if len(results) != 2 {
		return "", fmt.Errorf("unexpected amount of results during screen name extraction of uri %s", uri)
	}

	return results[1], nil
}
