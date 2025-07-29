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
		ModuleItems []*Tweet `json:"moduleItems"`
	} `json:"instructions"`
}

func (t *Timeline) TombstoneEntries() (tweets []*Tweet) {
	if t == nil {
		return
	}

	for _, instruction := range t.Instructions {
		if instruction.Type != "TimelineAddEntries" &&
			instruction.Type != "TimelineAddToModule" {
			continue
		}

		for _, entry := range instruction.Entries {
			if entry.Content.Items == nil {
				continue
			}

			for _, item := range entry.Content.Items {
				if item.Item.ItemContent.TweetResults.Result.Tombstone != nil {
					tweets = append(tweets, item)
				}
			}
		}

		for _, moduleItem := range instruction.ModuleItems {
			if moduleItem.Item.ItemContent.TweetResults.Result.Tombstone != nil {
				tweets = append(tweets, moduleItem)
			}
		}
	}

	return tweets
}

// TweetEntries returns all tweet entries from the entries in the timeline response (it also returns cursor entries)
func (t *Timeline) TweetEntries(userIDs ...string) (tweets []*Tweet) {
	if t == nil {
		return tweets
	}

	for _, instruction := range t.Instructions {
		if instruction.Type != "TimelineAddEntries" &&
			instruction.Type != "TimelineAddToModule" {
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

		// some Twitter user items starting with page 2 return tweets as module items
		// absolutely no idea why exactly (example user ID: 2323917366)
		for _, moduleItem := range instruction.ModuleItems {
			// possibly blocked tweet in your region or you got blocked
			if moduleItem.Item.ItemContent.TweetResults.Result.TweetData() == nil {
				continue
			}

			if len(userIDs) != 0 {
				inAllowedUsers := false
				for _, userID := range userIDs {
					if moduleItem.Item.ItemContent.TweetResults.Result.TweetData().Core.UserResults.Result != nil &&
						userID == moduleItem.Item.ItemContent.TweetResults.Result.TweetData().Core.UserResults.Result.RestID.String() {
						inAllowedUsers = true
						break
					}
				}

				// not in allowed users, skip entry (most likely advertisement entries)
				if !inAllowedUsers {
					continue
				}
			}

			tweets = append(tweets, moduleItem)
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
