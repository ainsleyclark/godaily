// Copyright (c) 2026 godaily (Ainsley Clark)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package candidates

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	socialgw "github.com/ainsleyclark/godaily/pkg/gateway/social"
	socialsvc "github.com/ainsleyclark/godaily/pkg/services/social"
)

// promoCycleLen is the M-M-C pattern length. Two meetups for each
// conference. With 52 Wednesdays/year this gives ~35 meetup slots and
// ~17 conference slots — enough to cover every entry in both pools at
// least once a year.
const promoCycleLen = 3

// communityTemplates is the per-platform pool of message bodies. Every
// template leads with {{.Mention}} so the tag is the first thing the
// reader and the platform's auto-tagger both see.
//
// {{.Mention}} resolves to:
//   - Bluesky/Mastodon: "@handle" when configured (auto-tags the
//     account), or {{.Name}} when not.
//   - LinkedIn:         the company URL when configured (LinkedIn
//     renders it as a clickable card and surfaces the share in the
//     company's mentions feed), or {{.Name}} when not.
//
// Keep the strings short — Bluesky caps at 300 chars and we want
// headroom for long conference names.
var communityTemplates = map[socialgw.Platform][]string{
	socialgw.PlatformLinkedIn: {
		"{{.Mention}} — {{.Description}}\n\n{{.URL}}",
		"{{.Mention}} ({{.Location}}) — {{.Description}}\n\n{{.URL}}",
		"{{.Mention}}: {{.Description}}\n\n{{.URL}}",
	},
	socialgw.PlatformBluesky: {
		"{{.Mention}} — {{.Description}} {{.URL}}",
		"{{.Mention}} ({{.Location}}) — {{.Description}} {{.URL}}",
		"{{.Mention}}: {{.Description}} {{.URL}}",
	},
	socialgw.PlatformMastodon: {
		"{{.Mention}} — {{.Description}} #golang {{.URL}}",
		"{{.Mention}} ({{.Location}}) — {{.Description}} #golang {{.URL}}",
		"{{.Mention}}: {{.Description}} #golang {{.URL}}",
	},
}

// communityEntry is one row from conferences.yaml or meetups.yaml. Conference
// entries set EndDate ("YYYY-MM-DD") so the candidate can filter out past
// editions; meetups leave it blank.
type communityEntry struct {
	Slug        string           `yaml:"slug"`
	Name        string           `yaml:"name"`
	URL         string           `yaml:"url"`
	Location    string           `yaml:"location"`
	Description string           `yaml:"description"`
	Handles     communityHandles `yaml:"handles"`
	EndDate     string           `yaml:"end_date,omitempty"`
}

// communityHandles groups the social handles per platform. Each field
// holds the raw identifier (no leading @, no URL prefix); the candidate
// formats them for the wire when building post text.
type communityHandles struct {
	LinkedIn string `yaml:"linkedin"`
	Bluesky  string `yaml:"bluesky"`
	Mastodon string `yaml:"mastodon"`
}

// mentions returns the per-platform string to splice into the post body.
// Empty platforms are omitted so the template-render fallback (use Name)
// kicks in.
func (e communityEntry) mentions() map[socialgw.Platform]string {
	out := make(map[socialgw.Platform]string)
	if e.Handles.LinkedIn != "" {
		out[socialgw.PlatformLinkedIn] = "https://www.linkedin.com/company/" + e.Handles.LinkedIn
	}
	if e.Handles.Bluesky != "" {
		out[socialgw.PlatformBluesky] = "@" + e.Handles.Bluesky
	}
	if e.Handles.Mastodon != "" {
		out[socialgw.PlatformMastodon] = "@" + e.Handles.Mastodon
	}
	return out
}

// Community runs every Wednesday and tags a Go conference or meetup. The
// pool rotates 2:1 meetups-to-conferences (M-M-C…); within each pool the
// candidate walks entries in slug order and posts the first one not yet
// promoted this calendar year. If this week's pool is exhausted it falls
// through to the other pool so the slot doesn't go quiet.
type Community struct {
	conferences []communityEntry
	meetups     []communityEntry
	posts       news.SocialPostRepository
}

// NewCommunity parses the two embedded YAMLs and constructs the
// candidate. Panics on YAML parse error — these are compile-time-checked
// static assets, so a runtime error is the right escape hatch.
func NewCommunity(conferencesYAML, meetupsYAML []byte, posts news.SocialPostRepository) *Community {
	return &Community{
		conferences: mustParseCommunityYAML(conferencesYAML, "conferences"),
		meetups:     mustParseCommunityYAML(meetupsYAML, "meetups"),
		posts:       posts,
	}
}

func mustParseCommunityYAML(b []byte, label string) []communityEntry {
	var entries []communityEntry
	if err := yaml.Unmarshal(b, &entries); err != nil {
		panic(fmt.Sprintf("community: parse %s YAML: %v", label, err))
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Slug < entries[j].Slug })
	return entries
}

// Kind reports the candidate's SocialPostKind.
func (c *Community) Kind() news.SocialPostKind { return news.SocialPostKindCommunity }

// Eligible picks the right pool for this Wednesday and returns the first
// entry not yet promoted this year. Falls through to the other pool when
// this week's pool is exhausted.
func (c *Community) Eligible(ctx context.Context, now time.Time) (socialsvc.CandidateContext, bool, error) {
	today := now.UTC().Truncate(24 * time.Hour)
	idx := weekIndex(now)
	year := now.UTC().Year()

	primary, secondary := c.poolsForWeek(idx, today)

	for _, pool := range [][]communityEntry{primary, secondary} {
		for _, entry := range pool {
			subject := fmt.Sprintf("community:%s:%d", entry.Slug, year)
			posted, err := c.posts.HasPostedBySubject(ctx, subject, platformAnchor)
			if err != nil {
				return socialsvc.CandidateContext{}, false, errors.Wrap(err, "checking community subject")
			}
			if posted {
				continue
			}
			return socialsvc.CandidateContext{
				Kind:     c.Kind(),
				Subject:  subject,
				URL:      entry.URL,
				Mentions: entry.mentions(),
				Payload:  communityPayload{Entry: entry, WeekIndex: idx},
			}, true, nil
		}
	}
	return socialsvc.CandidateContext{}, false, nil
}

// Generate renders one of the per-platform templates with the entry's
// mention spliced in (or its plain name as fallback when no handle is
// configured for the platform).
func (c *Community) Generate(_ context.Context, _ ai.Prompter, platform socialgw.Platform, cctx socialsvc.CandidateContext) (string, error) {
	payload, ok := cctx.Payload.(communityPayload)
	if !ok {
		return "", errors.New("community: payload missing")
	}
	pool := communityTemplates[platform]
	if len(pool) == 0 {
		return "", nil
	}

	mention := payload.Entry.Name
	if m := cctx.Mentions[platform]; m != "" {
		mention = m
	}

	tpl := pool[mod(payload.WeekIndex, len(pool))]
	return renderTemplate(tpl, payload.Entry, mention), nil
}

// communityPayload travels through CandidateContext from Eligible to
// Generate so per-platform rendering doesn't re-derive the week index.
type communityPayload struct {
	Entry     communityEntry
	WeekIndex int
}

// poolsForWeek returns (primary, secondary) for the given week-index.
// Conferences are filtered to entries whose end_date is today or later
// (past editions are dead data until the next year's entry is added).
// secondary acts as the fall-through pool when primary is exhausted.
func (c *Community) poolsForWeek(weekIdx int, today time.Time) (primary, secondary []communityEntry) {
	upcoming := upcomingConferences(c.conferences, today)
	if mod(weekIdx, promoCycleLen) == 2 {
		return upcoming, c.meetups
	}
	return c.meetups, upcoming
}

// upcomingConferences returns conferences whose EndDate is today or in
// the future. Empty EndDate means "always upcoming" (defensive — the
// schema only sets EndDate on conferences, but treating blank as live
// keeps the filter from accidentally hiding entries with malformed data).
func upcomingConferences(entries []communityEntry, today time.Time) []communityEntry {
	out := make([]communityEntry, 0, len(entries))
	for _, e := range entries {
		if e.EndDate == "" {
			out = append(out, e)
			continue
		}
		end, err := time.Parse("2006-01-02", e.EndDate)
		if err != nil {
			continue
		}
		if !end.Before(today) {
			out = append(out, e)
		}
	}
	return out
}

// weekIndex returns a monotonic integer that increments by exactly one
// every Wednesday-to-Wednesday. Each 7-day Unix bucket contains exactly
// one Wednesday, so consecutive Wednesdays land in consecutive buckets.
func weekIndex(t time.Time) int {
	return int(t.UTC().Unix() / (7 * 24 * 60 * 60))
}

// mod is Go's % that always returns a non-negative result, so negative
// weekIndex values (theoretically possible with a clock skew) still pick
// a valid slice index.
func mod(a, b int) int {
	r := a % b
	if r < 0 {
		r += b
	}
	return r
}

// renderTemplate substitutes the three placeholders without bringing in
// text/template — keeps the candidate dependency-free and the templates
// readable as plain strings.
func renderTemplate(tpl string, e communityEntry, mention string) string {
	return strings.NewReplacer(
		"{{.Mention}}", mention,
		"{{.Name}}", e.Name,
		"{{.Description}}", e.Description,
		"{{.URL}}", e.URL,
		"{{.Location}}", e.Location,
	).Replace(tpl)
}
