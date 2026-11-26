// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package candidates

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/gateway/meetup"
	"github.com/ainsleyclark/godaily/pkg/services/social/candidate"
)

// promoCycleLen is the M-M-C pattern length: two meetup weeks for each
// conference week. The week's pool is only a preference; when it has
// nothing timely the other pool is tried, and when neither does the slot
// is skipped rather than filled with an undated directory entry.
const promoCycleLen = 3

const (
	// eventWindowMin / eventWindowMax bound how far ahead a meetup event
	// must be to be promoted: far enough out that readers can still RSVP,
	// close enough that "this is happening soon" is the hook.
	eventWindowMin = 3 * 24 * time.Hour
	eventWindowMax = 21 * 24 * time.Hour

	// conferenceWindowMin / conferenceWindowMax are the equivalent for
	// conferences, which need more lead time (travel, tickets).
	conferenceWindowMin = 7 * 24 * time.Hour
	conferenceWindowMax = 42 * 24 * time.Hour

	// groupCooldown stops one active group from being promoted every
	// time it schedules an event.
	groupCooldown = 8 * 7 * 24 * time.Hour

	// fetchConcurrency caps parallel Meetup feed requests.
	fetchConcurrency = 5

	// maxEventTitleRunes keeps event posts inside Bluesky's 300-char cap
	// with room for the group name, date, location and URL.
	maxEventTitleRunes = 100
)

// EventSource returns a Meetup group's upcoming events. Satisfied by
// *meetup.Client; tests use a fake.
type EventSource interface {
	UpcomingEvents(ctx context.Context, group string) ([]meetup.Event, error)
}

// eventTemplates is the per-platform pool for meetup-event posts and
// conferenceTemplates the pool for conference posts. Every template leads
// with {{.Mention}} so the tag is the first thing the reader and the
// platform's auto-tagger both see.
//
// {{.Mention}} resolves to:
//   - Bluesky/Mastodon: "@handle" when configured (auto-tags the
//     account), or {{.Name}} when not.
//   - LinkedIn:         {{.Name}} when a URN is configured (the
//     LinkedIn poster turns it into a real inline-mention tag via
//     commentaryAnnotations), the company URL when only a slug is
//     configured (renders as a clickable card), or {{.Name}} when
//     neither is set.
//
// Keep the strings short — Bluesky caps at 300 chars.
//
// Separators are plain hyphens, never em dashes — em dashes are off-brand
// (see aiutil.SanitisePost, which also strips them at the publish boundary
// as a backstop for the AI-generated kinds).
var (
	eventTemplates = map[social.Platform][]string{
		social.LinkedIn: {
			"{{.Mention}} - next meetup {{.EventDate}} in {{.Location}}: \"{{.EventTitle}}\"\n\n{{.EventURL}}",
			"{{.Mention}} ({{.Location}}) - \"{{.EventTitle}}\", {{.EventDate}}.\n\n{{.EventURL}}",
			"{{.Mention}}: \"{{.EventTitle}}\" - {{.EventDate}}, {{.Location}}.\n\n{{.EventURL}}",
		},
		social.Bluesky: {
			"{{.Mention}} - next meetup {{.EventDate}} in {{.Location}}: \"{{.EventTitle}}\" {{.EventURL}}",
			"{{.Mention}} ({{.Location}}) - \"{{.EventTitle}}\", {{.EventDate}}. {{.EventURL}}",
			"{{.Mention}}: \"{{.EventTitle}}\" - {{.EventDate}}, {{.Location}}. {{.EventURL}}",
		},
		social.Mastodon: {
			"{{.Mention}} - next meetup {{.EventDate}} in {{.Location}}: \"{{.EventTitle}}\" #golang {{.EventURL}}",
			"{{.Mention}} ({{.Location}}) - \"{{.EventTitle}}\", {{.EventDate}}. #golang {{.EventURL}}",
			"{{.Mention}}: \"{{.EventTitle}}\" - {{.EventDate}}, {{.Location}}. #golang {{.EventURL}}",
		},
	}

	conferenceTemplates = map[social.Platform][]string{
		social.LinkedIn: {
			"{{.Mention}} is {{.Countdown}} - {{.Dates}}, {{.Location}}. {{.Description}}\n\n{{.URL}}",
			"{{.Mention}} ({{.Location}}, {{.Dates}}) - {{.Description}}\n\n{{.URL}}",
		},
		social.Bluesky: {
			"{{.Mention}} is {{.Countdown}} - {{.Dates}}, {{.Location}}. {{.Description}} {{.URL}}",
			"{{.Mention}} ({{.Location}}, {{.Dates}}) - {{.Description}} {{.URL}}",
		},
		social.Mastodon: {
			"{{.Mention}} is {{.Countdown}} - {{.Dates}}, {{.Location}}. {{.Description}} #golang {{.URL}}",
			"{{.Mention}} ({{.Location}}, {{.Dates}}) - {{.Description}} #golang {{.URL}}",
		},
	}
)

// communityEntry is one row from conferences.yaml or meetups.yaml.
// Conference entries set StartDate/EndDate ("YYYY-MM-DD") so posts can
// carry the dates and go out while the conference is weeks away. Meetup
// entries set MeetupGroup so the candidate can read the group's upcoming
// events; a blank MeetupGroup means the group has no readable feed and is
// never promoted.
type communityEntry struct {
	Slug        string           `yaml:"slug"`
	Name        string           `yaml:"name"`
	URL         string           `yaml:"url"`
	Location    string           `yaml:"location"`
	Description string           `yaml:"description"`
	Handles     communityHandles `yaml:"handles"`
	StartDate   string           `yaml:"start_date,omitempty"`
	EndDate     string           `yaml:"end_date,omitempty"`
	MeetupGroup string           `yaml:"meetup_group,omitempty"`
}

// communityHandles groups the social handles per platform. Bluesky and
// Mastodon hold the raw identifier (no leading @, no URL prefix); LinkedIn
// is a nested object so we can carry both the public-page slug and the
// numeric org URN.
type communityHandles struct {
	LinkedIn linkedInHandle `yaml:"linkedin"`
	Bluesky  string         `yaml:"bluesky"`
	Mastodon string         `yaml:"mastodon"`
}

// linkedInHandle carries the two LinkedIn identifiers we care about.
// Slug is the public /company/<slug> path used for the URL-card fallback.
// URN is "urn:li:organization:<id>" used by the inline-mention annotator
// to render a real @-tag in the post body.
type linkedInHandle struct {
	Slug string `yaml:"slug"`
	URN  string `yaml:"urn"`
}

// mentions returns the per-platform identifier to splice into the post
// body. Empty platforms are omitted so the template-render fallback
// (use Name) kicks in.
//
// LinkedIn has two paths:
//   - URN set: Handle carries the urn:li:organization, DisplayName carries
//     the entry name so the LinkedIn poster's annotation pipeline can find
//     the substring in commentary and emit a real inline-mention tag.
//   - URN blank, slug set: Handle carries the public company URL with no
//     DisplayName — LinkedIn renders it as a clickable card but it isn't
//     an annotated mention.
func (e communityEntry) mentions() []social.Mention {
	var out []social.Mention
	switch {
	case e.Handles.LinkedIn.URN != "":
		out = append(out, social.Mention{
			Platform:    social.LinkedIn,
			Handle:      e.Handles.LinkedIn.URN,
			DisplayName: e.Name,
		})
	case e.Handles.LinkedIn.Slug != "":
		out = append(out, social.Mention{
			Platform: social.LinkedIn,
			Handle:   "https://www.linkedin.com/company/" + e.Handles.LinkedIn.Slug,
		})
	}
	if e.Handles.Bluesky != "" {
		out = append(out, social.Mention{
			Platform: social.Bluesky,
			Handle:   "@" + e.Handles.Bluesky,
		})
	}
	if e.Handles.Mastodon != "" {
		out = append(out, social.Mention{
			Platform: social.Mastodon,
			Handle:   "@" + e.Handles.Mastodon,
		})
	}
	return out
}

// Community runs every Wednesday and tags a Go meetup with an event
// coming up, or a Go conference that is a few weeks away. The pool
// preference rotates 2:1 meetups-to-conferences (M-M-C…); if this week's
// pool has nothing timely it falls through to the other pool, and if
// neither does the slot is skipped.
//
// Meetups are picked from their live Meetup.com calendars (via
// EventSource): the soonest event 3-21 days out whose group hasn't been
// promoted in the last 8 weeks. Conferences are picked from
// conferences.yaml: the soonest one starting 7-42 days out that hasn't
// been promoted this year.
type Community struct {
	conferences []communityEntry
	meetups     []communityEntry
	posts       social.PostRepository
	events      EventSource
}

// NewCommunity parses the two embedded YAMLs and constructs the
// candidate. events may be nil, in which case no meetup is ever eligible.
// Panics on YAML parse error — these are compile-time-checked static
// assets, so a runtime error is the right escape hatch.
func NewCommunity(conferencesYAML, meetupsYAML []byte, posts social.PostRepository, events EventSource) *Community {
	return &Community{
		conferences: mustParseCommunityYAML(conferencesYAML, "conferences"),
		meetups:     mustParseCommunityYAML(meetupsYAML, "meetups"),
		posts:       posts,
		events:      events,
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
func (c *Community) Kind() social.PostKind { return social.PostKindCommunity }

// Eligible picks this Wednesday's preferred pool, then the other, and
// returns the first timely entry. Returns false when neither pool has a
// meetup event or conference inside its window.
func (c *Community) Eligible(ctx context.Context, now time.Time) (candidate.CandidateContext, bool, error) {
	now = now.UTC()
	idx := weekIndex(now)

	pickers := []func(context.Context, time.Time, int) (candidate.CandidateContext, bool, error){
		c.pickMeetupEvent,
		c.pickConference,
	}
	if mod(idx, promoCycleLen) == 2 {
		pickers[0], pickers[1] = pickers[1], pickers[0]
	}

	for _, pick := range pickers {
		cctx, ok, err := pick(ctx, now, idx)
		if err != nil || ok {
			return cctx, ok, err
		}
	}
	return candidate.CandidateContext{}, false, nil
}

// upcomingEvent pairs a meetup entry with one of its events.
type upcomingEvent struct {
	entry communityEntry
	event meetup.Event
}

// pickMeetupEvent returns the soonest in-window meetup event that hasn't
// been posted and whose group is outside its cool-down.
func (c *Community) pickMeetupEvent(ctx context.Context, now time.Time, idx int) (candidate.CandidateContext, bool, error) {
	if c.events == nil {
		return candidate.CandidateContext{}, false, nil
	}

	from, to := now.Add(eventWindowMin), now.Add(eventWindowMax)
	var inWindow []upcomingEvent
	for _, u := range c.fetchUpcoming(ctx) {
		if !u.event.Start.Before(from) && !u.event.Start.After(to) {
			inWindow = append(inWindow, u)
		}
	}
	sort.SliceStable(inWindow, func(i, j int) bool {
		a, b := inWindow[i], inWindow[j]
		if !a.event.Start.Equal(b.event.Start) {
			return a.event.Start.Before(b.event.Start)
		}
		return a.entry.Slug < b.entry.Slug
	})

	for _, u := range inWindow {
		subject := fmt.Sprintf("community:%s:event:%s", u.entry.Slug, eventKey(u.event.UID))
		posted, err := c.posts.HasPostedBySubject(ctx, subject, platformAnchor)
		if err != nil {
			return candidate.CandidateContext{}, false, errors.Wrap(err, "checking community event subject")
		}
		if posted {
			continue
		}
		recent, err := c.posts.HasPostedSubjectPrefixSince(ctx, communityPrefix(u.entry.Slug), platformAnchor, now.Add(-groupCooldown))
		if err != nil {
			return candidate.CandidateContext{}, false, errors.Wrap(err, "checking community group cool-down")
		}
		if recent {
			continue
		}

		ev := u.event
		url := ev.URL
		if url == "" {
			url = u.entry.URL
		}
		return candidate.CandidateContext{
			Kind:     c.Kind(),
			Subject:  subject,
			URL:      url,
			Mentions: u.entry.mentions(),
			Payload:  communityPayload{Entry: u.entry, WeekIndex: idx, Event: &ev},
		}, true, nil
	}
	return candidate.CandidateContext{}, false, nil
}

// fetchUpcoming reads every meetup's calendar concurrently. A group whose
// feed fails is logged and treated as having no events: the feed is not
// a supported API, so one bad response must never fail the slot.
func (c *Community) fetchUpcoming(ctx context.Context) []upcomingEvent {
	var (
		mu  sync.Mutex
		wg  sync.WaitGroup
		out []upcomingEvent
		sem = make(chan struct{}, fetchConcurrency)
	)
	for _, entry := range c.meetups {
		if entry.MeetupGroup == "" {
			continue
		}
		wg.Add(1)
		go func(entry communityEntry) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			events, err := c.events.UpcomingEvents(ctx, entry.MeetupGroup)
			if err != nil {
				slog.WarnContext(ctx, "Community: reading meetup calendar failed",
					"slug", entry.Slug, "group", entry.MeetupGroup, "error", err)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			for _, ev := range events {
				out = append(out, upcomingEvent{entry: entry, event: ev})
			}
		}(entry)
	}
	wg.Wait()
	return out
}

// pickConference returns the soonest conference starting inside the
// conference window that hasn't been promoted this year.
func (c *Community) pickConference(ctx context.Context, now time.Time, idx int) (candidate.CandidateContext, bool, error) {
	year := now.Year()
	for _, uc := range upcomingConferences(c.conferences, now) {
		subject := fmt.Sprintf("community:%s:%d", uc.entry.Slug, year)
		posted, err := c.posts.HasPostedBySubject(ctx, subject, platformAnchor)
		if err != nil {
			return candidate.CandidateContext{}, false, errors.Wrap(err, "checking community subject")
		}
		if posted {
			continue
		}
		return candidate.CandidateContext{
			Kind:     c.Kind(),
			Subject:  subject,
			URL:      uc.entry.URL,
			Mentions: uc.entry.mentions(),
			Payload: communityPayload{
				Entry:     uc.entry,
				WeekIndex: idx,
				Dates:     formatDateRange(uc.start, uc.end),
				Countdown: formatCountdown(now, uc.start),
			},
		}, true, nil
	}
	return candidate.CandidateContext{}, false, nil
}

// upcomingConference is a conference with its parsed dates.
type upcomingConference struct {
	entry      communityEntry
	start, end time.Time
}

// upcomingConferences returns conferences starting 7-42 days after now,
// soonest first. Entries without a parseable start_date are skipped —
// the whole point of the post is the date. A missing or bad end_date is
// treated as a one-day conference.
func upcomingConferences(entries []communityEntry, now time.Time) []upcomingConference {
	from, to := now.Add(conferenceWindowMin), now.Add(conferenceWindowMax)
	out := make([]upcomingConference, 0, len(entries))
	for _, e := range entries {
		start, err := time.Parse(time.DateOnly, e.StartDate)
		if err != nil {
			continue
		}
		end, err := time.Parse(time.DateOnly, e.EndDate)
		if err != nil || end.Before(start) {
			end = start
		}
		if start.Before(from) || start.After(to) {
			continue
		}
		out = append(out, upcomingConference{entry: e, start: start, end: end})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].start.Equal(out[j].start) {
			return out[i].start.Before(out[j].start)
		}
		return out[i].entry.Slug < out[j].entry.Slug
	})
	return out
}

// Generate renders one of the per-platform templates with the entry's
// mention spliced in (or its plain name as fallback when no handle is
// configured for the platform).
func (c *Community) Generate(_ context.Context, _ ai.Prompter, p social.Platform, cctx candidate.CandidateContext) (string, error) {
	payload, ok := cctx.Payload.(communityPayload)
	if !ok {
		return "", errors.New("community: payload missing")
	}

	templates := conferenceTemplates
	if payload.Event != nil {
		templates = eventTemplates
	}
	pool := templates[p]
	if len(pool) == 0 {
		return "", nil
	}

	mention := payload.Entry.Name
	if m := cctx.Mention(p); m != "" && !strings.HasPrefix(m, "urn:li:") {
		mention = m
	}

	tpl := pool[mod(payload.WeekIndex, len(pool))]
	return renderTemplate(tpl, payload, mention), nil
}

// communityPayload travels through CandidateContext from Eligible to
// Generate so per-platform rendering doesn't re-derive anything. Event is
// set for meetup posts; Dates and Countdown for conference posts.
type communityPayload struct {
	Entry     communityEntry
	WeekIndex int
	Event     *meetup.Event
	Dates     string
	Countdown string
}

// communityPrefix is the subject prefix shared by every post about slug,
// whatever its suffix (year for conferences and legacy posts, event ID
// for meetup events).
func communityPrefix(slug string) string {
	return "community:" + slug + ":"
}

// eventKey turns a Meetup UID ("event_317454295@meetup.com") into the
// short, stable key used in the subject ("317454295"). Other UID shapes
// pass through with only the domain stripped.
func eventKey(uid string) string {
	key, _, _ := strings.Cut(uid, "@")
	return strings.TrimPrefix(key, "event_")
}

// formatEventDate renders the event start in its own time zone, e.g.
// "Thu 3 Dec, 18:00", or "Thu 3 Dec" for all-day events.
func formatEventDate(ev meetup.Event) string {
	if ev.AllDay {
		return ev.Start.Format("Mon 2 Jan")
	}
	return ev.Start.Format("Mon 2 Jan, 15:04")
}

// formatDateRange renders a conference's dates compactly: "23 Oct",
// "1-3 Nov" or "30 Sep - 2 Oct".
func formatDateRange(start, end time.Time) string {
	switch {
	case !end.After(start):
		return start.Format("2 Jan")
	case start.Month() == end.Month() && start.Year() == end.Year():
		return fmt.Sprintf("%d-%s", start.Day(), end.Format("2 Jan"))
	default:
		return start.Format("2 Jan") + " - " + end.Format("2 Jan")
	}
}

// formatCountdown renders how far away start is: "in 10 days" under two
// weeks, otherwise whole weeks ("in 3 weeks").
func formatCountdown(now, start time.Time) string {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	days := int(start.Sub(today).Hours() / 24)
	if days < 14 {
		return fmt.Sprintf("in %d days", days)
	}
	return fmt.Sprintf("in %d weeks", days/7)
}

// truncateWords shortens s to at most limit runes, cutting at the last
// word boundary and appending "...".
func truncateWords(s string, limit int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= limit {
		return string(r)
	}
	cut := string(r[:limit-3])
	if i := strings.LastIndex(cut, " "); i > 0 {
		cut = cut[:i]
	}
	return strings.TrimRight(cut, " ,.;:-") + "..."
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

// renderTemplate substitutes the placeholders without bringing in
// text/template — keeps the candidate dependency-free and the templates
// readable as plain strings.
func renderTemplate(tpl string, p communityPayload, mention string) string {
	e := p.Entry
	pairs := []string{
		"{{.Mention}}", mention,
		"{{.Name}}", e.Name,
		"{{.Description}}", e.Description,
		"{{.URL}}", e.URL,
		"{{.Location}}", e.Location,
		"{{.Dates}}", p.Dates,
		"{{.Countdown}}", p.Countdown,
	}
	if p.Event != nil {
		eventURL := p.Event.URL
		if eventURL == "" {
			eventURL = e.URL
		}
		pairs = append(pairs,
			"{{.EventTitle}}", truncateWords(p.Event.Title, maxEventTitleRunes),
			"{{.EventDate}}", formatEventDate(*p.Event),
			"{{.EventURL}}", eventURL,
		)
	}
	return strings.NewReplacer(pairs...).Replace(tpl)
}
