// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slackdata

import (
	"strings"
	"testing"
	"time"

	slackgo "github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
)

func TestKindLabel(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "Featured", kindLabel(social.PostKindFeatured))
	assert.Equal(t, "New source", kindLabel(social.PostKindNewSource))
	assert.Empty(t, kindLabel(""))
}

func TestPostPublished_NothingLive(t *testing.T) {
	t.Parallel()
	_, ok := PostPublished(social.PostKindFeatured, "x", nil, []social.PostResult{
		{Platform: social.Bluesky, Skipped: true},
	})
	assert.False(t, ok)
}

// TestPostPublished_PerPlatform asserts that distinct copy renders one
// section per platform with text + a "View on" button.
func TestPostPublished_PerPlatform(t *testing.T) {
	t.Parallel()
	id := int64(42)
	req, ok := PostPublished(social.PostKindFeatured, "Go 1.30 lands", &id, []social.PostResult{
		{Platform: social.Bluesky, Text: "bluesky copy", PostURL: "https://bsky.app/a"},
		{Platform: social.LinkedIn, Text: "linkedin copy", PostURL: "https://linkedin.com/b"},
	})
	require.True(t, ok)

	msg := flatten(req)
	assert.Contains(t, msg, "Social post published: Featured")
	assert.Contains(t, msg, "Go 1.30 lands")
	assert.Contains(t, msg, "bluesky copy")
	assert.Contains(t, msg, "linkedin copy")
	assert.Contains(t, msg, "https://bsky.app/a")
	assert.Contains(t, msg, "https://linkedin.com/b")
	assert.Contains(t, msg, "Posted to 2 platforms")
}

// TestPostPublished_CollapsesIdenticalText asserts the card collapses to a
// single quote plus a button row when every platform shares the same copy.
func TestPostPublished_CollapsesIdenticalText(t *testing.T) {
	t.Parallel()
	req, ok := PostPublished(social.PostKindRecap, "Weekly recap", nil, []social.PostResult{
		{Platform: social.Bluesky, Text: "same copy", PostURL: "https://bsky.app/a"},
		{Platform: social.LinkedIn, Text: "same copy", PostURL: "https://linkedin.com/b"},
		{Platform: social.Mastodon, Text: "same copy", PostURL: "https://mastodon.social/c"},
	})
	require.True(t, ok)

	var actions, quotes int
	for _, blk := range req.Blocks.BlockSet {
		switch v := blk.(type) {
		case *slackgo.ActionBlock:
			actions++
			assert.Len(t, v.Elements.ElementSet, 3, "one button per platform")
		case *slackgo.SectionBlock:
			if v.Text != nil && strings.Contains(v.Text.Text, "same copy") {
				quotes++
			}
		}
	}
	assert.Equal(t, 1, actions)
	assert.Equal(t, 1, quotes)
}

func TestDraftsPublished(t *testing.T) {
	t.Parallel()
	date := time.Date(2026, time.May, 20, 0, 0, 0, 0, time.UTC)
	req, ok := DraftsPublished(date, []social.PostResult{
		{Kind: social.PostKindFeatured, Platform: social.Bluesky, Text: "publish me", PostURL: "https://bsky.app/abc"},
	})
	require.True(t, ok)

	msg := flatten(req)
	assert.Contains(t, msg, "Social drafts published")
	assert.Contains(t, msg, "Featured  ·  Bluesky")
	assert.Contains(t, msg, "publish me")
	assert.Contains(t, msg, "https://bsky.app/abc")
	assert.Contains(t, msg, "2026-05-20")
}

// flatten concatenates header/section/context text + accessory and action
// button URLs into one string.
func flatten(req slack.Request) string {
	var b strings.Builder
	b.WriteString(req.Text)
	for _, blk := range req.Blocks.BlockSet {
		switch v := blk.(type) {
		case *slackgo.SectionBlock:
			if v.Text != nil {
				b.WriteString("\n")
				b.WriteString(v.Text.Text)
			}
			if v.Accessory != nil && v.Accessory.ButtonElement != nil {
				b.WriteString("\n")
				b.WriteString(v.Accessory.ButtonElement.URL)
			}
		case *slackgo.HeaderBlock:
			if v.Text != nil {
				b.WriteString("\n")
				b.WriteString(v.Text.Text)
			}
		case *slackgo.ContextBlock:
			for _, el := range v.ContextElements.Elements {
				if t, ok := el.(*slackgo.TextBlockObject); ok {
					b.WriteString("\n")
					b.WriteString(t.Text)
				}
			}
		case *slackgo.ActionBlock:
			for _, el := range v.Elements.ElementSet {
				if btn, ok := el.(*slackgo.ButtonBlockElement); ok {
					b.WriteString("\n")
					b.WriteString(btn.URL)
				}
			}
		}
	}
	return b.String()
}
