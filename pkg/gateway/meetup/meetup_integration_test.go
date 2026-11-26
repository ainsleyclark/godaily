// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build integration

package meetup_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/ainsleyclark/godaily/pkg/data"
	"github.com/ainsleyclark/godaily/pkg/gateway/meetup"
)

// TestMeetup_Integration fetches the live iCal feed of every group in
// meetups.yaml. A failure here means a group was renamed, removed or made
// private on Meetup.com (fix its meetup_group), or Meetup changed the
// feed.
func TestMeetup_Integration(t *testing.T) {
	var groups []struct {
		Slug        string `yaml:"slug"`
		MeetupGroup string `yaml:"meetup_group"`
	}
	require.NoError(t, yaml.Unmarshal(data.Meetups, &groups))

	client := meetup.New()
	var total int
	t.Cleanup(func() { t.Logf("%d upcoming events across all groups", total) })

	for _, g := range groups {
		if g.MeetupGroup == "" {
			continue
		}
		t.Run(g.Slug, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			events, err := client.UpcomingEvents(ctx, g.MeetupGroup)
			require.NoError(t, err)
			total += len(events)

			for _, ev := range events {
				t.Logf("%s  %s  %s", ev.Start.Format(time.RFC3339), ev.Title, ev.URL)
				assert.NotEmpty(t, ev.UID)
				assert.NotEmpty(t, ev.Title)
				assert.Contains(t, ev.URL, "meetup.com/")
				assert.False(t, ev.Start.IsZero())
				assert.True(t, ev.Start.After(time.Now().Add(-48*time.Hour)), "feed should only list upcoming events")
				if !ev.AllDay {
					assert.NotEqual(t, "Local", ev.Start.Location().String(), "start must not depend on the host zone")
				}
			}
		})
	}
}
