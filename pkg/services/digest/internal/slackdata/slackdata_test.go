// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slackdata

import (
	"strings"
	"testing"

	slackgo "github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
)

func TestBuildSummary(t *testing.T) {
	t.Parallel()

	t.Run("Header + subject + intro + items render", func(t *testing.T) {
		t.Parallel()

		req := BuildSummary(Summary{
			IssueDate: "2026-06-01",
			IssueID:   42,
			Subject:   "GoDaily June 1",
			Intro:     "A handful of generics improvements landed.",
			ItemCount: 12,
		})

		flat := flatten(req)
		assert.Contains(t, flat, "Digest ready for review")
		assert.Contains(t, flat, "2026-06-01")
		assert.Contains(t, flat, "GoDaily June 1")
		assert.Contains(t, flat, "A handful of generics improvements landed.")
		assert.Contains(t, flat, "12")
	})

	t.Run("View issue button uses dashboard URL with issue id", func(t *testing.T) {
		t.Parallel()

		req := BuildSummary(Summary{IssueDate: "2026-06-01", IssueID: 42, Subject: "x"})
		assert.Contains(t, flatten(req), "/issues/42")
		assert.Contains(t, flatten(req), env.DashboardURL)
	})

	t.Run("Each draft renders kind, platform, text and an Edit button", func(t *testing.T) {
		t.Parallel()

		req := BuildSummary(Summary{
			IssueDate: "2026-06-01",
			IssueID:   42,
			Drafts: []Draft{
				{ID: 1, Kind: "featured", Platform: "bluesky", Text: "draft body for bluesky"},
				{ID: 2, Kind: "featured", Platform: "linkedin", Text: "draft body for linkedin"},
				{ID: 3, Kind: "recap", Platform: "bluesky", Text: "this week’s top clicks"},
			},
		})
		flat := flatten(req)

		assert.Contains(t, flat, "Featured · Bluesky")
		assert.Contains(t, flat, "Featured · Linkedin")
		assert.Contains(t, flat, "Recap · Bluesky")
		assert.Contains(t, flat, "draft body for bluesky")
		assert.Contains(t, flat, "this week’s top clicks")

		assert.Contains(t, flat, "/social/drafts?id=1")
		assert.Contains(t, flat, "/social/drafts?id=2")
		assert.Contains(t, flat, "/social/drafts?id=3")
	})

	t.Run("Drafts are emitted in stable kind+platform order", func(t *testing.T) {
		t.Parallel()

		req := BuildSummary(Summary{
			IssueDate: "2026-06-01",
			Drafts: []Draft{
				{ID: 3, Kind: "recap", Platform: "linkedin", Text: "r-li"},
				{ID: 1, Kind: "featured", Platform: "bluesky", Text: "f-bs"},
				{ID: 2, Kind: "featured", Platform: "linkedin", Text: "f-li"},
			},
		})
		flat := flatten(req)
		i1 := strings.Index(flat, "f-bs")
		i2 := strings.Index(flat, "f-li")
		i3 := strings.Index(flat, "r-li")
		require.True(t, i1 < i2 && i2 < i3, "drafts not in stable order: %d %d %d", i1, i2, i3)
	})

	t.Run("Footer mentions auto-publish window", func(t *testing.T) {
		t.Parallel()
		req := BuildSummary(Summary{IssueDate: "2026-06-01"})
		assert.Contains(t, flatten(req), "Auto-publishes")
	})

	t.Run("Attachment is Info-coloured", func(t *testing.T) {
		t.Parallel()
		req := BuildSummary(Summary{IssueDate: "2026-06-01"})
		require.Len(t, req.Attachments, 1)
		assert.Equal(t, slack.ColorInfo, req.Attachments[0].Color)
	})
}

// flatten concatenates every block's text + accessory button into one
// string so we can run a simple Contains over the rendered message.
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
				btn := v.Accessory.ButtonElement
				b.WriteString("\n")
				if btn.Text != nil {
					b.WriteString(btn.Text.Text)
					b.WriteString(" ")
				}
				b.WriteString(btn.URL)
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
		}
	}
	return b.String()
}
