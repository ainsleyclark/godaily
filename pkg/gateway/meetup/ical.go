// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package meetup

import (
	"io"
	"sort"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
	"github.com/pkg/errors"
)

// Parse reads an RFC 5545 calendar and returns its VEVENTs, soonest
// first. Parsing (line unfolding, TEXT unescaping, nested components,
// TZID resolution) is delegated to github.com/arran4/golang-ical; this
// function only maps the fields GoDaily uses onto Event.
//
// Cancelled events and events without a UID or a parseable start are
// dropped rather than failing the whole feed.
func Parse(r io.Reader) ([]Event, error) {
	cal, err := ics.ParseCalendarWithOptions(r,
		ics.WithWindowsTimezoneMapping(),
		ics.WithUnknownPropertyHandler(keepTrailingCalendarProperty),
	)
	if err != nil {
		return nil, errors.Wrap(err, "parsing calendar")
	}

	events := make([]Event, 0, len(cal.Events()))
	for _, ve := range cal.Events() {
		ev, ok := toEvent(ve)
		if ok {
			events = append(events, ev)
		}
	}

	sort.SliceStable(events, func(i, j int) bool { return events[i].Start.Before(events[j].Start) })
	return events, nil
}

// keepTrailingCalendarProperty accepts calendar-level properties that
// appear after the first component. Meetup's feed ends with
// X-ORIGINAL-URL and X-WR-CALNAME after the last VEVENT, which RFC 5545
// doesn't allow and the library's default (strict) handler rejects —
// failing every feed that has at least one event.
func keepTrailingCalendarProperty(cal *ics.Calendar, _ string, p *ics.BaseProperty) error {
	cal.CalendarProperties = append(cal.CalendarProperties, ics.CalendarProperty{BaseProperty: *p})
	return nil
}

// toEvent maps a VEVENT onto Event, reporting false for events that
// should be skipped.
func toEvent(ve *ics.VEvent) (Event, bool) {
	uid := ve.Id()
	if uid == "" || strings.EqualFold(propValue(ve, ics.ComponentPropertyStatus), "CANCELLED") {
		return Event{}, false
	}

	start, allDay, ok := startOf(ve)
	if !ok {
		return Event{}, false
	}

	return Event{
		UID:    uid,
		Title:  propValue(ve, ics.ComponentPropertySummary),
		Start:  start,
		AllDay: allDay,
		URL:    propValue(ve, ics.ComponentPropertyUrl),
		Venue:  propValue(ve, ics.ComponentPropertyLocation),
	}, true
}

// startOf returns the event's DTSTART, normalised so results don't depend
// on the host's local time zone:
//
//   - TZID times stay in that zone, so formatting shows local wall-clock time.
//   - UTC ("Z") times stay in UTC.
//   - All-day (VALUE=DATE) and floating times (no TZID, no Z) are read as
//     UTC; the library would otherwise use time.Local.
//   - An unknown TZID falls back to UTC rather than dropping the event —
//     the date is still right, only the wall-clock time may be off.
func startOf(ve *ics.VEvent) (time.Time, bool, bool) {
	prop := ve.GetProperty(ics.ComponentPropertyDtStart)
	if prop == nil {
		return time.Time{}, false, false
	}

	if isDateValue(prop) {
		t, err := ve.GetAllDayStartAt()
		if err != nil {
			return time.Time{}, false, false
		}
		return asUTC(t), true, true
	}

	_, hasTZID := prop.ICalParameters[string(ics.ParameterTzid)]
	t, err := ve.GetStartAt()
	switch {
	case err == nil && (hasTZID || strings.HasSuffix(prop.Value, "Z")):
		return t, false, true
	case err == nil:
		return asUTC(t), false, true // floating time
	case hasTZID:
		// Unknown zone: read the wall-clock value as UTC.
		if ft, ferr := time.Parse("20060102T150405", strings.TrimSpace(prop.Value)); ferr == nil {
			return ft, false, true
		}
	}
	return time.Time{}, false, false
}

// isDateValue reports whether DTSTART is a bare date (VALUE=DATE, or an
// 8-digit value with no time part).
func isDateValue(prop *ics.IANAProperty) bool {
	for _, v := range prop.ICalParameters[string(ics.ParameterValue)] {
		if strings.EqualFold(v, string(ics.ValueDataTypeDate)) {
			return true
		}
	}
	return len(strings.TrimSpace(prop.Value)) == len("20060102")
}

// asUTC keeps t's wall-clock fields and reinterprets them in UTC.
func asUTC(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.UTC)
}

// propValue returns a property's (already unescaped) value, or "".
func propValue(ve *ics.VEvent, p ics.ComponentProperty) string {
	if prop := ve.GetProperty(p); prop != nil {
		return strings.TrimSpace(prop.Value)
	}
	return ""
}
