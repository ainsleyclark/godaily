// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package meetup_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/pkg/gateway/meetup"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	require.NoError(t, err)
	return b
}

func TestParse(t *testing.T) {
	t.Run("Parses events soonest first and drops invalid ones", func(t *testing.T) {
		events, err := meetup.Parse(strings.NewReader(string(readFixture(t, "events.ics"))))
		require.NoError(t, err)

		// Cancelled, unparseable-start and UID-less events are dropped.
		require.Len(t, events, 3)
		assert.Equal(t, "event_100000001@meetup.com", events[0].UID)
		assert.Equal(t, "event_300000003@meetup.com", events[1].UID)
		assert.Equal(t, "event_200000002@meetup.com", events[2].UID)
	})

	t.Run("Resolves TZID to the event's local time", func(t *testing.T) {
		events, err := meetup.Parse(strings.NewReader(string(readFixture(t, "events.ics"))))
		require.NoError(t, err)

		got := events[0]
		assert.Equal(t, "Europe/Berlin", got.Start.Location().String())
		assert.Equal(t, 18, got.Start.Hour(), "wall-clock hour must be local, not UTC")
		assert.Equal(t, time.Date(2026, 12, 3, 17, 0, 0, 0, time.UTC), got.Start.UTC())
		assert.False(t, got.AllDay)
	})

	t.Run("Unfolds continuation lines", func(t *testing.T) {
		events, err := meetup.Parse(strings.NewReader(string(readFixture(t, "events.ics"))))
		require.NoError(t, err)

		got := events[0]
		assert.Equal(t, "Building AI-Native Backends in Go: Spec-Compliant MCP Servers with mcp-go", got.Title)
		assert.Equal(t, "https://www.meetup.com/example-gophers/events/100000001/", got.URL)
		assert.Equal(t, "Example Office, Munich", got.Venue)
	})

	t.Run("Ignores properties on nested components", func(t *testing.T) {
		events, err := meetup.Parse(strings.NewReader(string(readFixture(t, "events.ics"))))
		require.NoError(t, err)
		assert.NotContains(t, events[0].Title, "Reminder")
		assert.Equal(t, 2026, events[0].Start.Year(), "VALARM DTSTART must not overwrite the event's")
	})

	t.Run("Unescapes text and parses UTC and all-day starts", func(t *testing.T) {
		events, err := meetup.Parse(strings.NewReader(string(readFixture(t, "events.ics"))))
		require.NoError(t, err)

		allDay := events[1]
		assert.True(t, allDay.AllDay)
		assert.Equal(t, time.Date(2026, 12, 10, 0, 0, 0, 0, time.UTC), allDay.Start)

		utc := events[2]
		assert.Equal(t, "January talks, pizza; and demos", utc.Title)
		assert.Equal(t, time.UTC, utc.Start.Location())
		assert.Equal(t, time.Date(2027, 1, 14, 17, 0, 0, 0, time.UTC), utc.Start)
	})

	t.Run("Empty calendar returns no events", func(t *testing.T) {
		events, err := meetup.Parse(strings.NewReader(string(readFixture(t, "empty.ics"))))
		require.NoError(t, err)
		assert.Empty(t, events)
	})

	t.Run("Unknown TZID falls back to UTC", func(t *testing.T) {
		cal := "BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\nUID:x\r\nDTSTART;TZID=Mars/Olympus:20261203T180000\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"
		events, err := meetup.Parse(strings.NewReader(cal))
		require.NoError(t, err)
		require.Len(t, events, 1)
		assert.Equal(t, time.Date(2026, 12, 3, 18, 0, 0, 0, time.UTC), events[0].Start)
	})

	t.Run("Handles bare LF line endings and quoted params", func(t *testing.T) {
		cal := "BEGIN:VCALENDAR\nBEGIN:VEVENT\nUID:y\nDTSTART;TZID=\"Europe/Madrid\":20261202T190000\nSUMMARY:Hola\nEND:VEVENT\nEND:VCALENDAR\n"
		events, err := meetup.Parse(strings.NewReader(cal))
		require.NoError(t, err)
		require.Len(t, events, 1)
		assert.Equal(t, "Europe/Madrid", events[0].Start.Location().String())
		assert.Equal(t, "Hola", events[0].Title)
	})
}

func TestClient_UpcomingEvents(t *testing.T) {
	t.Run("Fetches the group's iCal feed", func(t *testing.T) {
		var gotPath, gotUA string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath, gotUA = r.URL.Path, r.UserAgent()
			w.Header().Set("Content-Type", "text/calendar")
			_, _ = w.Write(readFixture(t, "events.ics"))
		}))
		defer srv.Close()

		c := meetup.New(meetup.WithBaseURL(srv.URL+"/"), meetup.WithHTTPClient(srv.Client()))
		events, err := c.UpcomingEvents(context.Background(), "example-gophers")
		require.NoError(t, err)
		assert.Len(t, events, 3)
		assert.Equal(t, "/example-gophers/events/ical/", gotPath)
		assert.Contains(t, gotUA, "godaily")
	})

	t.Run("Non-200 is an error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		defer srv.Close()

		c := meetup.New(meetup.WithBaseURL(srv.URL))
		_, err := c.UpcomingEvents(context.Background(), "blocked")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "403")
	})

	t.Run("Rejects invalid group names without a request", func(t *testing.T) {
		c := meetup.New(meetup.WithBaseURL("http://127.0.0.1:1"))
		for _, g := range []string{"", "  ", "a/b", "a?b"} {
			_, err := c.UpcomingEvents(context.Background(), g)
			assert.Error(t, err, g)
		}
	})

	t.Run("Transport error is returned", func(t *testing.T) {
		c := meetup.New(meetup.WithBaseURL("http://127.0.0.1:1"))
		_, err := c.UpcomingEvents(context.Background(), "x")
		assert.Error(t, err)
	})
}
