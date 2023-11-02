package graphql_api

type Timeline struct {
	Instructions []TimelineInstruction `json:"instructions"`
}

// TweetEntries returns all tweet entries from the entries in the timeline response (it also returns cursor entries)
func (t *Timeline) TweetEntries(userIDs ...string) (tweets []*Tweet) {
	for _, instruction := range t.Instructions {
		if instruction.Type != "TimelineAddEntries" {
			continue
		}

		for _, entry := range instruction.Entries {
			if entry.Content.EntryType != "TimelineTimelineItem" {
				continue
			}

			if len(userIDs) != 0 {
				inAllowedUsers := false
				for _, userID := range userIDs {
					if entry.Content.ItemContent.TweetResults.Result.Core.UserResults.Result != nil &&
						userID == entry.Content.ItemContent.TweetResults.Result.Core.UserResults.Result.RestID.String() {
						inAllowedUsers = true
						break
					}
				}

				// not in allowed users, skip entry (most likely advertisement entries)
				if !inAllowedUsers {
					continue
				}
			}

			tweets = append(tweets, entry)
		}
	}

	return tweets
}

func (t *Timeline) BottomCursor() string {
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
