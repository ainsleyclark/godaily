// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package meetup

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Parse reads an RFC 5545 calendar and returns its VEVENTs, soonest
// first. It understands only what Meetup's feed uses: UID, SUMMARY,
// DTSTART (with TZID, UTC "Z" or VALUE=DATE), URL, LOCATION and STATUS.
// Cancelled events and events without a UID or a parseable start are
// dropped rather than failing the whole feed.
func Parse(r io.Reader) ([]Event, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, errors.Wrap(err, "reading calendar")
	}

	var (
		events []Event
		stack  []string // open component names, innermost last
		cur    Event
		status string
		valid  bool
	)

	for _, line := range unfold(string(raw)) {
		name, params, value, ok := splitProperty(line)
		if !ok {
			continue
		}

		switch name {
		case "BEGIN":
			stack = append(stack, strings.ToUpper(value))
			if strings.EqualFold(value, "VEVENT") {
				cur, status, valid = Event{}, "", true
			}
			continue
		case "END":
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			if strings.EqualFold(value, "VEVENT") {
				if valid && cur.UID != "" && !cur.Start.IsZero() && !strings.EqualFold(status, "CANCELLED") {
					events = append(events, cur)
				}
			}
			continue
		}

		// Only properties directly on a VEVENT count; nested components
		// (VALARM) carry their own DTSTART/SUMMARY we must ignore.
		if len(stack) == 0 || stack[len(stack)-1] != "VEVENT" {
			continue
		}

		switch name {
		case "UID":
			cur.UID = value
		case "SUMMARY":
			cur.Title = unescapeText(value)
		case "URL":
			cur.URL = value
		case "LOCATION":
			cur.Venue = unescapeText(value)
		case "STATUS":
			status = value
		case "DTSTART":
			start, allDay, err := parseDateTime(value, params)
			if err != nil {
				valid = false
				continue
			}
			cur.Start, cur.AllDay = start, allDay
		}
	}

	sort.SliceStable(events, func(i, j int) bool { return events[i].Start.Before(events[j].Start) })
	return events, nil
}

// unfold normalises line endings and joins folded continuation lines
// (RFC 5545 §3.1: a line starting with a space or tab continues the
// previous one, minus that single leading character).
func unfold(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	var out []string
	for _, l := range strings.Split(s, "\n") {
		if (strings.HasPrefix(l, " ") || strings.HasPrefix(l, "\t")) && len(out) > 0 {
			out[len(out)-1] += l[1:]
			continue
		}
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}

// splitProperty splits "NAME;P1=V1;P2=V2:value" into its parts. The
// value starts at the first colon outside a quoted parameter value.
// Parameter names are upper-cased; the property name is too.
func splitProperty(line string) (name string, params map[string]string, value string, ok bool) {
	inQuote := false
	colon := -1
	for i, r := range line {
		switch {
		case r == '"':
			inQuote = !inQuote
		case r == ':' && !inQuote:
			colon = i
		}
		if colon >= 0 {
			break
		}
	}
	if colon < 0 {
		return "", nil, "", false
	}

	head, value := line[:colon], line[colon+1:]
	parts := strings.Split(head, ";")
	name = strings.ToUpper(strings.TrimSpace(parts[0]))
	if name == "" {
		return "", nil, "", false
	}

	params = make(map[string]string, len(parts)-1)
	for _, p := range parts[1:] {
		k, v, found := strings.Cut(p, "=")
		if !found {
			continue
		}
		params[strings.ToUpper(k)] = strings.Trim(v, `"`)
	}
	return name, params, value, true
}

// parseDateTime parses a DTSTART value. Supported forms:
//
//	20261203T180000Z               UTC
//	TZID=Europe/Berlin:20261203T180000   local time in TZID
//	20261203T180000                floating (treated as UTC)
//	VALUE=DATE:20261203            all-day
//
// An unknown TZID falls back to UTC rather than dropping the event — the
// date is still right, only the wall-clock time may be off.
func parseDateTime(value string, params map[string]string) (time.Time, bool, error) {
	value = strings.TrimSpace(value)

	if strings.EqualFold(params["VALUE"], "DATE") || len(value) == len("20060102") {
		t, err := time.Parse("20060102", value)
		if err != nil {
			return time.Time{}, false, fmt.Errorf("parsing date %q: %w", value, err)
		}
		return t, true, nil
	}

	if strings.HasSuffix(value, "Z") {
		t, err := time.Parse("20060102T150405Z", value)
		if err != nil {
			return time.Time{}, false, fmt.Errorf("parsing UTC time %q: %w", value, err)
		}
		return t, false, nil
	}

	loc := time.UTC
	if tzid := params["TZID"]; tzid != "" {
		if l, err := time.LoadLocation(tzid); err == nil {
			loc = l
		}
	}
	t, err := time.ParseInLocation("20060102T150405", value, loc)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("parsing local time %q: %w", value, err)
	}
	return t, false, nil
}

// unescapeText reverses RFC 5545 TEXT escaping (\\, \;, \, and \n).
func unescapeText(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i == len(s)-1 {
			b.WriteByte(s[i])
			continue
		}
		i++
		switch s[i] {
		case 'n', 'N':
			b.WriteByte('\n')
		default: // \\ \; \, and anything unknown: keep the literal char.
			b.WriteByte(s[i])
		}
	}
	return b.String()
}
