// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package data_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/ainsleyclark/godaily/pkg/data"
)

type entry struct {
	Slug        string  `yaml:"slug"`
	Name        string  `yaml:"name"`
	URL         string  `yaml:"url"`
	MeetupGroup *string `yaml:"meetup_group"`
	StartDate   string  `yaml:"start_date"`
	EndDate     string  `yaml:"end_date"`
}

func parse(t *testing.T, b []byte) []entry {
	t.Helper()
	var out []entry
	require.NoError(t, yaml.Unmarshal(b, &out))
	require.NotEmpty(t, out)
	return out
}

func TestMeetups(t *testing.T) {
	seen := map[string]bool{}
	for _, e := range parse(t, data.Meetups) {
		t.Run(e.Slug, func(t *testing.T) {
			assert.NotEmpty(t, e.Slug)
			assert.False(t, seen[e.Slug], "duplicate slug")
			seen[e.Slug] = true
			assert.NotEmpty(t, e.Name)

			// meetup_group must be present on every entry, even if blank,
			// so a missing feed is a deliberate choice.
			require.NotNil(t, e.MeetupGroup, "meetup_group must be set (use \"\" when not on Meetup.com)")
			group := *e.MeetupGroup

			onMeetup := strings.Contains(e.URL, "meetup.com/")
			if !onMeetup {
				assert.Empty(t, group, "meetup_group set for a group not on Meetup.com")
				return
			}
			if group == "" {
				return // deliberately no feed (see the entry's comment)
			}
			assert.Equal(t, strings.ToLower(group), group, "meetup_group should be lowercase")
			assert.NotContains(t, group, "/")
			assert.Contains(t, strings.ToLower(e.URL), "meetup.com/"+group+"/", "url and meetup_group disagree")
		})
	}
}

func TestConferences(t *testing.T) {
	seen := map[string]bool{}
	for _, e := range parse(t, data.Conferences) {
		t.Run(e.Slug, func(t *testing.T) {
			assert.False(t, seen[e.Slug], "duplicate slug")
			seen[e.Slug] = true

			start, err := time.Parse(time.DateOnly, e.StartDate)
			require.NoError(t, err, "start_date is required for the community rotation")
			end, err := time.Parse(time.DateOnly, e.EndDate)
			require.NoError(t, err)
			assert.False(t, end.Before(start), "end_date before start_date")
		})
	}
}
