package graphql_api

type Timeline struct {
	Instructions []struct {
		Type    string `json:"type"`
		Entries []struct {
			EntryID string `json:"entryId"`
			Content struct {
				EntryType  string   `json:"entryType"`
				Value      string   `json:"value"`
				CursorType string   `json:"cursorType"`
				Items      []*Tweet `json:"items"`
			} `json:"content"`
		} `json:"entries"`
	} `json:"instructions"`
}

// TweetEntries returns all tweet entries from the entries in the timeline response (it also returns cursor entries)
func (t *Timeline) TweetEntries(userIDs ...string) (tweets []*Tweet) {
	if t == nil {
		return tweets
	}

	for _, instruction := range t.Instructions {
		if instruction.Type != "TimelineAddEntries" {
			continue
		}

		for _, entry := range instruction.Entries {
			if entry.Content.Items == nil {
				continue
			}

			for _, item := range entry.Content.Items {
				// possibly blocked tweet in your region or you got blocked
				if item.Item.ItemContent.TweetResults.Result.TweetData() == nil {
					continue
				}

				if len(userIDs) != 0 {
					inAllowedUsers := false
					for _, userID := range userIDs {
						if item.Item.ItemContent.TweetResults.Result.TweetData().Core.UserResults.Result != nil &&
							userID == item.Item.ItemContent.TweetResults.Result.TweetData().Core.UserResults.Result.RestID.String() {
							inAllowedUsers = true
							break
						}
					}

					// not in allowed users, skip entry (most likely advertisement entries)
					if !inAllowedUsers {
						continue
					}
				}

				tweets = append(tweets, item)
			}
		}
	}

	return tweets
}

func (t *Timeline) BottomCursor() string {
	if t == nil {
		return ""
	}

	for _, instruction := range t.Instructions {
		if instruction.Type != "TimelineAddEntries" {
			continue
		}

		for _, entry := range instruction.Entries {
			if entry.Content.CursorType != "Bottom" {
				continue
			}

			return entry.Content.Value
		}
	}

	return ""
}
