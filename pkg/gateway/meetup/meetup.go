// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package meetup reads a Meetup.com group's upcoming events from the
// group's public iCal feed (https://www.meetup.com/<group>/events/ical/).
//
// The feed is the "subscribe to calendar" link Meetup offers every group
// page. Unlike Meetup's GraphQL API it needs no account, OAuth app or
// Meetup Pro subscription, but it is also not a supported API: Meetup can
// change or remove it without notice. Callers should treat an error or an
// empty result as "no upcoming event" and degrade gracefully.
package meetup

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	// Embeds the IANA zone database so TZID parameters resolve on
	// minimal production images that ship without /usr/share/zoneinfo.
	_ "time/tzdata"

	"github.com/pkg/errors"
)

const (
	// DefaultBaseURL is the Meetup.com origin the feed is served from.
	DefaultBaseURL = "https://www.meetup.com"

	// userAgent identifies GoDaily to Meetup, mirroring the Reddit source.
	userAgent = "godaily/1.0 (+https://godaily.dev)"

	// defaultTimeout bounds a single feed request.
	defaultTimeout = 10 * time.Second

	// maxBodyBytes caps the response read. A busy group's feed is a few
	// tens of KB; anything larger is not a calendar we want to parse.
	maxBodyBytes = 2 << 20
)

// Event is one upcoming event from a group's calendar.
type Event struct {
	// UID is the calendar UID, e.g. "event_317454295@meetup.com". Stable
	// across feed refreshes, so it is safe to key idempotency off.
	UID string
	// Title is the event's SUMMARY.
	Title string
	// Start is the event start in the event's own time zone (TZID), so
	// formatting it shows local wall-clock time.
	Start time.Time
	// AllDay is true when DTSTART was a bare date (VALUE=DATE).
	AllDay bool
	// URL is the event's own page.
	URL string
	// Venue is the event's LOCATION, when the organiser set one.
	Venue string
}

// Client fetches group calendars.
type Client struct {
	http    *http.Client
	baseURL string
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient overrides the HTTP client (tests, custom transports).
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.http = hc }
}

// WithBaseURL overrides the Meetup origin (tests).
func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(u, "/") }
}

// New constructs a Client with a 10s timeout.
func New(opts ...Option) *Client {
	c := &Client{
		http:    &http.Client{Timeout: defaultTimeout},
		baseURL: DefaultBaseURL,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// UpcomingEvents returns the group's upcoming events, soonest first. The
// group is the path segment of the group page, e.g. "golang-amsterdam"
// for https://www.meetup.com/golang-amsterdam/. Meetup only publishes
// future events in the feed, so an empty slice means nothing is
// currently scheduled.
func (c *Client) UpcomingEvents(ctx context.Context, group string) ([]Event, error) {
	group = strings.TrimSpace(group)
	if group == "" || strings.ContainsAny(group, "/?#") {
		return nil, fmt.Errorf("meetup: invalid group %q", group)
	}

	u := c.baseURL + "/" + url.PathEscape(group) + "/events/ical/"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return nil, errors.Wrap(err, "meetup: building request")
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/calendar")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "meetup: fetching %s", group)
	}
	defer resp.Body.Close() //nolint:errcheck // read-only body

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("meetup: fetching %s: unexpected status %d", group, resp.StatusCode)
	}

	events, err := Parse(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, errors.Wrapf(err, "meetup: parsing %s", group)
	}
	return events, nil
}
