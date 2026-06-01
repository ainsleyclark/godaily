// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slackcard

import (
	"strings"
	"testing"

	slackgo "github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
)

func TestBlockquote(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "> one\n> two", blockquote("one\ntwo"))
	assert.Equal(t, "> trimmed", blockquote("  trimmed  "))

	long := strings.Repeat("a", maxText+50)
	got := blockquote(long)
	assert.True(t, strings.HasSuffix(got, "…"), "long text should be truncated with an ellipsis")
	assert.LessOrEqual(t, len([]rune(strings.TrimPrefix(got, "> "))), maxText+1)
}

func TestSection_Accessory(t *testing.T) {
	t.Parallel()
	blk := section(Row{
		Heading: "Bluesky",
		Text:    "hello world",
		Button:  &slack.LinkButton{Label: "View on Bluesky", URL: "https://bsky.app/x", Style: "primary"},
	})
	sec, ok := blk.(*slackgo.SectionBlock)
	require.True(t, ok)
	assert.Equal(t, "*Bluesky*\n> hello world", sec.Text.Text)
	require.NotNil(t, sec.Accessory)
	require.NotNil(t, sec.Accessory.ButtonElement)
	assert.Equal(t, "https://bsky.app/x", sec.Accessory.ButtonElement.URL)
	assert.Equal(t, slackgo.Style("primary"), sec.Accessory.ButtonElement.Style)
}

func TestBuild_Structure(t *testing.T) {
	t.Parallel()
	req := Build("Title", "context line", "fallback text", slack.ColorSuccess,
		[]Row{{Heading: "Bluesky", Text: "copy", Button: &slack.LinkButton{Label: "View", URL: "https://x"}}},
		Context("trailing line"),
	)

	assert.Equal(t, "fallback text", req.Text)
	require.Len(t, req.Attachments, 1)
	assert.Equal(t, slack.ColorSuccess, req.Attachments[0].Color)

	// header, context, divider, one row, trailing context.
	require.Len(t, req.Blocks.BlockSet, 5)
	_, ok := req.Blocks.BlockSet[0].(*slackgo.HeaderBlock)
	assert.True(t, ok)
	_, ok = req.Blocks.BlockSet[2].(*slackgo.DividerBlock)
	assert.True(t, ok)
}
