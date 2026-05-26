// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"context"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/ainsleyclark/godaily/pkg/data"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
)

// conferencePhaseTags maps notify_date index to the corresponding Tag.
// Index 0 = announcement, 1 = reminder (~3 months out), 2 = alert (~1 week out).
var conferencePhaseTags = []news.Tag{
	news.TagConference,
	news.TagConferenceReminder,
	news.TagConferenceAlert,
}

// conferencePhaseSuffixes are appended to the conference URL to produce a
// distinct (url, tag) key per phase, preventing dedup conflicts across phases.
var conferencePhaseSuffixes = []string{"#announce", "#reminder", "#alert"}

func init() {
	news.Register(news.SourceConferences, func(_ env.Config) news.Fetcher {
		return NewConferences(data.Conferences)
	})
}

// Conferences emits major Go conference notifications based on a curated YAML
// file. It emits an item for each conference whose notify_date matches today,
// setting Published to a time inside yesterday's collect window so the item
// is picked up by the daily collection run.
type Conferences struct {
	conferences []conferenceEntry
	now         func() time.Time
}

var _ news.Fetcher = &Conferences{}

// NewConferences parses yamlData and returns a ready-to-use Conferences source.
func NewConferences(yamlData []byte) *Conferences {
	var entries []conferenceEntry
	if err := yaml.Unmarshal(yamlData, &entries); err != nil {
		panic(fmt.Sprintf("conferences: failed to parse YAML: %v", err))
	}
	return &Conferences{
		conferences: entries,
		now:         time.Now,
	}
}

// Fetch returns conference notification items whose notify_date matches today.
// Published is set to yesterday noon so the item falls inside the collect window
// (yesterday midnight → today midnight) during the current daily run.
func (c *Conferences) Fetch(_ context.Context) ([]news.Item, error) {
	today := c.now().UTC().Truncate(24 * time.Hour)
	publishedAt := today.AddDate(0, 0, -1).Add(12 * time.Hour)

	var items []news.Item
	for _, conf := range c.conferences {
		for i, nd := range conf.NotifyDates {
			if nd.UTC().Truncate(24 * time.Hour).Equal(today) {
				items = append(items, conf.toItem(i, publishedAt))
				break
			}
		}
	}
	return items, nil
}

// conferenceEntry mirrors a single entry in conferences.yaml.
type conferenceEntry struct {
	Slug        string           `yaml:"slug"`
	Name        string           `yaml:"name"`
	URL         string           `yaml:"url"`
	Location    string           `yaml:"location"`
	StartDate   conferenceDate   `yaml:"start_date"`
	EndDate     conferenceDate   `yaml:"end_date"`
	Description string           `yaml:"description"`
	ImageURL    string           `yaml:"image_url"`
	NotifyDates []conferenceDate `yaml:"notify_dates"`
}

// toItem converts a conference entry at phase index i into a news.Item.
func (e conferenceEntry) toItem(phase int, published time.Time) news.Item {
	tag := news.TagConference
	if phase < len(conferencePhaseTags) {
		tag = conferencePhaseTags[phase]
	}

	// Append a fragment so each phase produces a distinct (url, tag) pair in
	// the DB, preventing cross-phase dedup collisions.
	suffix := ""
	if phase < len(conferencePhaseSuffixes) {
		suffix = conferencePhaseSuffixes[phase]
	}

	startDate := e.StartDate.Time()
	snippet := fmt.Sprintf("%s · %s", e.Location, startDate.Format("2 January 2006"))
	if e.Description != "" {
		snippet = e.Description
	}

	return news.Item{
		Source:    news.SourceConferences,
		Title:     e.Name,
		URL:       e.URL + suffix,
		ImageURL:  e.ImageURL,
		Snippet:   snippet,
		Tag:       tag,
		Published: published,
	}
}

// conferenceDate is a date-only YAML value (YYYY-MM-DD).
type conferenceDate struct {
	t time.Time
}

func (d *conferenceDate) UnmarshalYAML(value *yaml.Node) error {
	t, err := time.Parse("2006-01-02", value.Value)
	if err != nil {
		return fmt.Errorf("conferences: invalid date %q (expected YYYY-MM-DD): %w", value.Value, err)
	}
	d.t = t
	return nil
}

func (d conferenceDate) Time() time.Time {
	return d.t
}

func (d conferenceDate) UTC() time.Time {
	return d.t.UTC()
}
