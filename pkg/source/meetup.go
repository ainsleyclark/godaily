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
	"strings"
	"sync"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/source/ingest"
)

// meetupGroups is the curated list of Meetup.com group slugs for Go communities.
// Add a slug here to include a new group; remove one to stop fetching it.
// All groups are verified Go communities; no keyword filtering is needed.
var meetupGroups = []string{
	// UK
	"londongophers",
	// USA East
	"golang-nyc",
	"boston-golang",
	// USA West
	"golangsf",
	// USA Central
	"golang-chicago",
	// Canada
	"golang-toronto",
	// Europe
	"golang-berlin",
	"golang-paris",
	"golangamsterdam",
	// Asia-Pacific
	"golangsg",
	"golang-bangalore",
}

const meetupBaseURL = "https://www.meetup.com/"

// Meetup fetches upcoming Go meetup events from a curated set of Meetup.com groups.
type Meetup struct {
	groupURLs []string // full page URLs, one per group
}

var _ news.Fetcher = &Meetup{}

func init() {
	news.Register(news.SourceMeetup, func(cfg env.Config) news.Fetcher { return NewMeetup(cfg) })
}

// NewMeetup creates a Meetup source using the curated group list.
func NewMeetup(_ env.Config) *Meetup {
	urls := make([]string, len(meetupGroups))
	for i, slug := range meetupGroups {
		urls[i] = meetupBaseURL + slug + "/"
	}
	return &Meetup{groupURLs: urls}
}

// Fetch retrieves upcoming events from all configured Go meetup groups concurrently.
// Groups that fail to load are skipped so one bad group does not block the rest.
func (m *Meetup) Fetch(ctx context.Context) ([]news.Item, error) {
	results := make(chan []meetupEventItem, len(m.groupURLs))

	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup

	for _, u := range m.groupURLs {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			items, err := m.fetchGroup(ctx, u)
			if err != nil {
				results <- nil
				return
			}
			results <- items
		}(u)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var all []meetupEventItem
	for batch := range results {
		all = append(all, batch...)
	}

	return ingest.TransformAll(ctx, all), nil
}

// fetchGroup fetches and parses upcoming events from a single Meetup.com group page URL.
func (m *Meetup) fetchGroup(ctx context.Context, groupURL string) ([]meetupEventItem, error) {
	doc, err := ingest.FetchHTML(ctx, groupURL, "meetup")
	if err != nil {
		return nil, err
	}

	script := doc.Find(`script#__NEXT_DATA__`).Text()
	if script == "" {
		return nil, fmt.Errorf("meetup: no __NEXT_DATA__ at %s", groupURL)
	}

	var nd meetupNextData
	if err := json.Unmarshal([]byte(script), &nd); err != nil {
		return nil, fmt.Errorf("meetup: unmarshal %s: %w", groupURL, err)
	}

	state := nd.Props.PageProps.ApolloState

	var items []meetupEventItem
	for key, raw := range state {
		if !strings.HasPrefix(key, "Event:") {
			continue
		}
		var evt meetupEvent
		if err := json.Unmarshal(raw, &evt); err != nil {
			continue
		}
		var venue meetupVenue
		if ref := evt.Venue.Ref; ref != "" {
			if vRaw, ok := state[ref]; ok {
				_ = json.Unmarshal(vRaw, &venue)
			}
		}
		var photo meetupPhotoInfo
		if ref := evt.DisplayPhoto.Ref; ref != "" {
			if pRaw, ok := state[ref]; ok {
				_ = json.Unmarshal(pRaw, &photo)
			}
		}
		items = append(items, meetupEventItem{evt: evt, venue: venue, photo: photo})
	}

	return items, nil
}

type meetupEventItem struct {
	evt   meetupEvent
	venue meetupVenue
	photo meetupPhotoInfo
}

func (i meetupEventItem) ShouldInclude() bool {
	return i.evt.Status == "ACTIVE" && !strings.HasPrefix(i.evt.Title, "[Outside Event]")
}
func (i meetupEventItem) EnrichmentURL() string { return "" }

func (i meetupEventItem) Transform() news.Item {
	loc := i.venue.City
	if i.venue.Country != "" {
		loc += ", " + strings.ToUpper(i.venue.Country)
	}
	if i.evt.IsOnline || strings.TrimSpace(loc) == "" || loc == ", " {
		loc = "Online"
	}

	var parts []string
	if !i.evt.DateTime.IsZero() {
		parts = append(parts, i.evt.DateTime.Format("Mon Jan 2"))
	}
	if loc != "" {
		parts = append(parts, loc)
	}
	if i.evt.Going.TotalCount > 0 {
		parts = append(parts, fmt.Sprintf("%d RSVPs", i.evt.Going.TotalCount))
	}

	return news.Item{
		Source:    news.SourceMeetup,
		Title:     i.evt.Title,
		URL:       i.evt.EventURL,
		ImageURL:  i.photo.HighResURL,
		Snippet:   strings.Join(parts, " · "),
		Tag:       news.TagEvent,
		Published: time.Now().UTC(),
	}
}

// meetupNextData mirrors the shape of the __NEXT_DATA__ JSON embedded in
// Meetup.com group pages (Next.js + Apollo Client SSR).
type meetupNextData struct {
	Props struct {
		PageProps struct {
			ApolloState map[string]json.RawMessage `json:"__APOLLO_STATE__"`
		} `json:"pageProps"`
	} `json:"props"`
}

// apolloRef represents an Apollo Client normalised cache reference.
type apolloRef struct {
	Ref string `json:"__ref"`
}

type meetupEvent struct {
	Title       string    `json:"title"`
	EventURL    string    `json:"eventUrl"`
	Description string    `json:"description"`
	DateTime    time.Time `json:"dateTime"`
	// Status is "ACTIVE" for upcoming events, "PAST" for past events.
	Status string `json:"status"`
	Going  struct {
		TotalCount int `json:"totalCount"`
	} `json:"going"`
	Venue        apolloRef `json:"venue"`
	DisplayPhoto apolloRef `json:"displayPhoto"`
	IsOnline     bool      `json:"isOnline"`
}

type meetupVenue struct {
	Name    string `json:"name"`
	City    string `json:"city"`
	Country string `json:"country"`
}

type meetupPhotoInfo struct {
	HighResURL string `json:"highResUrl"`
}
