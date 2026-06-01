// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"context"
	"strings"
	"testing"

	slacksdk "github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
)

func TestKindLabel(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "Featured", kindLabel(social.PostKindFeatured))
	assert.Equal(t, "New source", kindLabel(social.PostKindNewSource))
	assert.Empty(t, kindLabel(""))
}

// TestNotifySuccess_PerPlatform asserts that when platforms carry distinct
// copy each renders its own section with text + a "View on" button.
func TestNotifySuccess_PerPlatform(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	var msg string
	f.slack.EXPECT().MustSend(gomock.Any(), gomock.Any()).
		Do(func(_ context.Context, req slack.Request) { msg = flattenSlackRequest(req) })

	id := int64(42)
	pc := publishCtx{kind: social.PostKindFeatured, subject: "Go 1.30 lands", issueID: &id}
	results := []social.PostResult{
		{Platform: social.Bluesky, Text: "bluesky copy", PostURL: "https://bsky.app/a"},
		{Platform: social.LinkedIn, Text: "linkedin copy", PostURL: "https://linkedin.com/b"},
	}
	f.service().notifySuccess(t.Context(), pc, results)

	assert.Contains(t, msg, "Social post published: Featured")
	assert.Contains(t, msg, "Go 1.30 lands")
	assert.Contains(t, msg, "bluesky copy")
	assert.Contains(t, msg, "linkedin copy")
	assert.Contains(t, msg, "https://bsky.app/a")
	assert.Contains(t, msg, "https://linkedin.com/b")
	assert.Contains(t, msg, "Posted to 2 platforms")
}

// TestNotifySuccess_CollapsesIdenticalText asserts the card collapses to a
// single quote plus a button row when every platform shares the same copy.
func TestNotifySuccess_CollapsesIdenticalText(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	var req slack.Request
	f.slack.EXPECT().MustSend(gomock.Any(), gomock.Any()).
		Do(func(_ context.Context, r slack.Request) { req = r })

	pc := publishCtx{kind: social.PostKindRecap, subject: "Weekly recap"}
	results := []social.PostResult{
		{Platform: social.Bluesky, Text: "same copy", PostURL: "https://bsky.app/a"},
		{Platform: social.LinkedIn, Text: "same copy", PostURL: "https://linkedin.com/b"},
		{Platform: social.Mastodon, Text: "same copy", PostURL: "https://mastodon.social/c"},
	}
	f.service().notifySuccess(t.Context(), pc, results)

	// Exactly one action block (the button row) and exactly one section
	// carrying the shared quote — the copy is not repeated per platform.
	var actions, quotes int
	for _, blk := range req.Blocks.BlockSet {
		switch v := blk.(type) {
		case *slacksdk.ActionBlock:
			actions++
			assert.Len(t, v.Elements.ElementSet, 3, "one button per platform")
		case *slacksdk.SectionBlock:
			if v.Text != nil && strings.Contains(v.Text.Text, "same copy") {
				quotes++
			}
		}
	}
	assert.Equal(t, 1, actions)
	assert.Equal(t, 1, quotes)
}
