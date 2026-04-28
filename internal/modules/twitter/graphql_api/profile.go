package graphql_api

import (
	"fmt"
	"strings"
	"time"
)

// FormatProfile renders a profile-view-style snapshot of the user, intended to be
// stored in the tracked item's generated_notes column so we keep a readable record
// of who an account was after the upstream profile is deleted or suspended.
//
// Output layout:
//
//	<Name>
//	@<screen_name>
//
//	<description with t.co URLs replaced by their expanded counterparts>
//
//	Location: <location>
//	Link: <url display_url>
//	Born: <Month> <Day>[, <Year>]
//	Joined: <Month> <Year>
//
// Empty sections are omitted; trailing newlines are trimmed.
func (u User) FormatProfile() string {
	var sb strings.Builder

	if u.Core.Name != "" {
		sb.WriteString(u.Core.Name)
		sb.WriteString("\n")
	}
	if u.Core.ScreenName != "" {
		sb.WriteString("@")
		sb.WriteString(u.Core.ScreenName)
		sb.WriteString("\n")
	}

	if u.Legacy.Description != "" {
		desc := u.Legacy.Description
		for _, ent := range u.Legacy.Entities.Description.URLs {
			if ent.URL != "" && ent.ExpandedURL != "" {
				desc = strings.ReplaceAll(desc, ent.URL, ent.ExpandedURL)
			}
		}
		sb.WriteString("\n")
		sb.WriteString(desc)
		sb.WriteString("\n")
	}

	var meta []string

	if u.Location.Location != "" {
		meta = append(meta, "Location: "+u.Location.Location)
	}

	if len(u.Legacy.Entities.URL.URLs) > 0 && u.Legacy.Entities.URL.URLs[0].DisplayURL != "" {
		meta = append(meta, "Link: "+u.Legacy.Entities.URL.URLs[0].DisplayURL)
	} else if u.Legacy.URL != "" {
		meta = append(meta, "Link: "+u.Legacy.URL)
	}

	if u.LegacyExtendedProfile.Birthdate != nil {
		b := u.LegacyExtendedProfile.Birthdate
		if b.Month >= 1 && b.Month <= 12 && b.Day >= 1 && b.Day <= 31 {
			line := fmt.Sprintf("Born: %s %d", time.Month(b.Month).String(), b.Day)
			if b.Year > 0 {
				line += fmt.Sprintf(", %d", b.Year)
			}
			meta = append(meta, line)
		}
	}

	if u.Core.CreatedAt != "" {
		if t, err := time.Parse(time.RubyDate, u.Core.CreatedAt); err == nil {
			meta = append(meta, fmt.Sprintf("Joined: %s %d", t.Month().String(), t.Year()))
		}
	}

	if len(meta) > 0 {
		sb.WriteString("\n")
		for _, line := range meta {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}

	return strings.TrimRight(sb.String(), "\n")
}
