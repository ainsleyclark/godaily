// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slackdata

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
	slacksdk "github.com/slack-go/slack"
)

func TestDeltaCount(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		curr, prev int64
		want       string
	}{
		"Both zero":           {curr: 0, prev: 0, want: "(–)"},
		"New from zero":       {curr: 5, prev: 0, want: "(new)"},
		"Negative from zero":  {curr: -3, prev: 0, want: "(-3)"},
		"Increase":            {curr: 110, prev: 100, want: "(↑ +10.0%)"},
		"Decrease":            {curr: 90, prev: 100, want: "(↓ -10.0%)"},
		"Unchanged":           {curr: 100, prev: 100, want: "(–)"},
		"Doubled":             {curr: 200, prev: 100, want: "(↑ +100.0%)"},
		"Halved":              {curr: 50, prev: 100, want: "(↓ -50.0%)"},
		"Fractional increase": {curr: 1027, prev: 1000, want: "(↑ +2.7%)"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, deltaCount(tc.curr, tc.prev))
		})
	}
}

func TestDeltaPoint(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		curr, prev float64
		want       string
	}{
		"Both zero":  {curr: 0, prev: 0, want: "(–)"},
		"Up":         {curr: 0.50, prev: 0.48, want: "(↑ +2.0pp)"},
		"Down":       {curr: 0.48, prev: 0.50, want: "(↓ -2.0pp)"},
		"Equal":      {curr: 0.5, prev: 0.5, want: "(–)"},
		"From zero":  {curr: 0.10, prev: 0, want: "(↑ +10.0pp)"},
		"To zero":    {curr: 0, prev: 0.10, want: "(↓ -10.0pp)"},
		"Small move": {curr: 0.501, prev: 0.500, want: "(↑ +0.1pp)"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, deltaPoint(tc.curr, tc.prev))
		})
	}
}

func TestFormatDelta(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		v    float64
		unit string
		want string
	}{
		"Positive percent": {v: 5.5, unit: "%", want: "(↑ +5.5%)"},
		"Negative percent": {v: -5.5, unit: "%", want: "(↓ -5.5%)"},
		"Zero":             {v: 0, unit: "%", want: "(–)"},
		"Positive pp":      {v: 2.1, unit: "pp", want: "(↑ +2.1pp)"},
		"Negative pp":      {v: -0.4, unit: "pp", want: "(↓ -0.4pp)"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, formatDelta(tc.v, tc.unit))
		})
	}
}

func TestSigned(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		in   int64
		want string
	}{
		"Positive": {in: 19, want: "+19"},
		"Zero":     {in: 0, want: "+0"},
		"Negative": {in: -5, want: "-5"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, signed(tc.in))
		})
	}
}

func TestHumanCount(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		in   int64
		want string
	}{
		"Zero":                {in: 0, want: "0"},
		"Single digit":        {in: 7, want: "7"},
		"Under 1k":            {in: 42, want: "42"},
		"Exactly 1000":        {in: 1000, want: "1,000"},
		"Thousands separator": {in: 1234, want: "1,234"},
		"Just under 10k":      {in: 9999, want: "9,999"},
		"Compact at 10k":      {in: 12345, want: "12.3k"},
		"Large compact":       {in: 1_234_567, want: "1234.6k"},
		"Negative under 10k":  {in: -1234, want: "-1,234"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, humanCount(tc.in))
		})
	}
}

func TestAddThousandsSep(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		in   int64
		want string
	}{
		"Zero":           {in: 0, want: "0"},
		"Below thousand": {in: 42, want: "42"},
		"Three digits":   {in: 999, want: "999"},
		"One thousand":   {in: 1000, want: "1,000"},
		"Four digits":    {in: 1234, want: "1,234"},
		"Six digits":     {in: 123456, want: "123,456"},
		"Seven digits":   {in: 1234567, want: "1,234,567"},
		"Negative":       {in: -1234, want: "-1,234"},
		"Negative small": {in: -42, want: "-42"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, addThousandsSep(tc.in))
		})
	}
}

func TestLastSubscriberPoint(t *testing.T) {
	t.Parallel()

	t.Run("Empty series", func(t *testing.T) {
		t.Parallel()
		_, ok := lastSubscriberPoint(engagement.SubscriberData{})
		assert.False(t, ok)
	})

	t.Run("Multiple points returns the last", func(t *testing.T) {
		t.Parallel()
		points := []engagement.SubscriberPoint{
			{ActiveAtEnd: 100},
			{ActiveAtEnd: 200},
			{ActiveAtEnd: 300},
		}
		got, ok := lastSubscriberPoint(engagement.SubscriberData{Points: points})
		assert.True(t, ok)
		assert.Equal(t, int64(300), got.ActiveAtEnd)
	})
}

func TestRoundup_LengthSanity(t *testing.T) {
	t.Parallel()
	// Even with full top-N lists, the message stays well under Slack's limit.
	data := RoundupData{
		From: time.Date(2026, 5, 17, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 5, 24, 0, 0, 0, 0, time.UTC),
		Summary: engagement.SummaryStats{
			IssuesSent: 7, Delivered: 1500, UniqueOpens: 700, UniqueClicks: 200,
			OpenRate: 0.5, ClickRate: 0.15,
		},
		Subs: engagement.SubscriberData{Points: []engagement.SubscriberPoint{{ActiveAtEnd: 1500, NetChange: 25, New: 30, Confirmed: 28, Unsubscribed: 5}}},
		Items: []engagement.ItemMetrics{
			{Title: strings.Repeat("X", 100), URL: "https://example.com/" + strings.Repeat("y", 80), Source: "src", Clicks: 50},
			{Title: strings.Repeat("X", 100), URL: "https://example.com/" + strings.Repeat("y", 80), Source: "src", Clicks: 40},
			{Title: strings.Repeat("X", 100), URL: "https://example.com/" + strings.Repeat("y", 80), Source: "src", Clicks: 30},
		},
		Tags:      []engagement.TagMetrics{{Tag: "a", Clicks: 1}, {Tag: "b", Clicks: 2}, {Tag: "c", Clicks: 3}},
		Sources:   []engagement.SourceMetrics{{Source: "x", Clicks: 1}, {Source: "y", Clicks: 2}, {Source: "z", Clicks: 3}},
		BestIssue: &engagement.IssueEngagement{Slug: "2026-05-22", ClickRate: 0.15, OpenRate: 0.5},
	}
	req := Roundup(data)
	assert.Less(t, len(flatten(req)), 4000, "message must fit in Slack's 4000-char limit")
}

func flatten(req slack.Request) string {
	var b strings.Builder
	b.WriteString(req.Text)
	for _, blk := range req.Blocks.BlockSet {
		switch v := blk.(type) {
		case *slacksdk.SectionBlock:
			if v.Text != nil {
				b.WriteString(v.Text.Text)
			}
			for _, f := range v.Fields {
				b.WriteString(f.Text)
			}
		case *slacksdk.HeaderBlock:
			if v.Text != nil {
				b.WriteString(v.Text.Text)
			}
		case *slacksdk.ContextBlock:
			for _, e := range v.ContextElements.Elements {
				if t, ok := e.(*slacksdk.TextBlockObject); ok {
					b.WriteString(t.Text)
				}
			}
		}
	}
	return b.String()
}
