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

package source

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/source/ingest"
)

// meetupSnippetMaxLen is the maximum rune length for a description-derived snippet.
const meetupSnippetMaxLen = 200

// meetupFallbacks are used when an event has no description.
// The index is chosen deterministically by hashing the event URL.
var meetupFallbacks = []string{
	"Join %s for an upcoming Go event in %s.",
	"Hosted by %s in %s — all are welcome.",
	"An upcoming Go community event from %s in %s.",
	"Come together with %s in %s.",
	"%s is hosting a Go meetup in %s.",
}

const meetupProURL = "https://www.meetup.com/pro/go/"

// Meetup fetches upcoming Go events from the official Go Developers Network Pro page.
// All 81 GDN-verified Go groups are covered from a single fetch.
type Meetup struct {
	proURL string
}

var _ news.Fetcher = &Meetup{}

func init() {
	news.Register(news.SourceMeetup, func(cfg env.Config) news.Fetcher { return NewMeetup(cfg) })
}

// NewMeetup creates a Meetup source using the Go Developers Network Pro page.
func NewMeetup(_ env.Config) *Meetup {
	return &Meetup{proURL: meetupProURL}
}

// Fetch retrieves upcoming Go events from the Go Developers Network Pro page.
func (m *Meetup) Fetch(ctx context.Context) ([]news.Item, error) {
	doc, err := ingest.FetchHTML(ctx, m.proURL, "meetup")
	if err != nil {
		return nil, err
	}

	script := doc.Find(`script#__NEXT_DATA__`).Text()
	if script == "" {
		return nil, fmt.Errorf("meetup: no __NEXT_DATA__ at %s", m.proURL)
	}

	var nd meetupProNextData
	if err := json.Unmarshal([]byte(script), &nd); err != nil {
		return nil, fmt.Errorf("meetup: unmarshal: %w", err)
	}

	events := nd.Props.PageProps.SEOData.Events
	items := make([]meetupProEventItem, len(events))
	for i, e := range events {
		items[i] = meetupProEventItem{evt: e}
	}

	return ingest.TransformAll(ctx, items), nil
}

type meetupProEventItem struct {
	evt meetupProEvent
}

func (i meetupProEventItem) ShouldInclude() bool {
	return !strings.HasPrefix(i.evt.Title, "[Outside Event]")
}

func (i meetupProEventItem) EnrichmentURL() string { return "" }

func (i meetupProEventItem) Transform() news.Item {
	// Prefer event venue city/country; fall back to the group city/country.
	city := i.evt.Group.City
	country := i.evt.Group.Country
	if i.evt.Venue != nil && i.evt.Venue.City != "" {
		city = i.evt.Venue.City
		country = i.evt.Venue.Country
	}

	loc := city
	if country != "" {
		loc += ", " + strings.ToUpper(country)
	}
	if i.evt.IsOnline || strings.TrimSpace(loc) == "" {
		loc = "Online"
	}

	return news.Item{
		Source:    news.SourceMeetup,
		Title:     i.evt.Title,
		URL:       i.evt.EventURL,
		ImageURL:  i.evt.DisplayPhoto.HighResURL,
		Snippet:   meetupSnippet(i.evt, loc),
		Tag:       news.TagEvent,
		Published: time.Now().UTC(),
	}
}

// meetupSnippet returns the best available snippet for an event. It prefers
// the first paragraph of the event description (truncated). When the
// description is blank it generates a varied fallback phrase.
func meetupSnippet(evt meetupProEvent, loc string) string {
	desc := strings.TrimSpace(evt.Description)
	if desc != "" {
		// Take the first paragraph (up to the first blank line).
		if idx := strings.Index(desc, "\n\n"); idx >= 0 {
			desc = strings.TrimSpace(desc[:idx])
		}
		// Flatten any remaining newlines.
		desc = strings.ReplaceAll(desc, "\n", " ")
		// Truncate at a word boundary.
		if utf8.RuneCountInString(desc) > meetupSnippetMaxLen {
			runes := []rune(desc)[:meetupSnippetMaxLen]
			for len(runes) > 0 && runes[len(runes)-1] != ' ' {
				runes = runes[:len(runes)-1]
			}
			desc = strings.TrimRight(string(runes), " ") + "…"
		}
		return desc
	}

	// No description — generate a varied fallback keyed on the event URL.
	h := fnv.New32a()
	_, _ = h.Write([]byte(evt.EventURL))
	tmpl := meetupFallbacks[h.Sum32()%uint32(len(meetupFallbacks))]
	return fmt.Sprintf(tmpl, evt.Group.Name, loc)
}

// meetupProNextData mirrors the __NEXT_DATA__ JSON embedded in the Meetup Pro
// network page (meetup.com/pro/go/).
type meetupProNextData struct {
	Props struct {
		PageProps struct {
			SEOData struct {
				Events []meetupProEvent `json:"events"`
			} `json:"SEOData"`
		} `json:"pageProps"`
	} `json:"props"`
}

type meetupProEvent struct {
	Title        string          `json:"title"`
	EventURL     string          `json:"eventUrl"`
	Description  string          `json:"description"`
	DateTime     time.Time       `json:"dateTime"`
	IsOnline     bool            `json:"isOnline"`
	DisplayPhoto meetupProPhoto  `json:"displayPhoto"`
	Group        meetupProGroup  `json:"group"`
	Venue        *meetupProVenue `json:"venue"`
	RSVPs        meetupProRSVPs  `json:"rsvps"`
}

type meetupProPhoto struct {
	HighResURL string `json:"highResUrl"`
}

type meetupProGroup struct {
	Name    string `json:"name"`
	City    string `json:"city"`
	Country string `json:"country"`
}

type meetupProVenue struct {
	City    string `json:"city"`
	Country string `json:"country"`
}

type meetupProRSVPs struct {
	TotalCount int `json:"totalCount"`
}
