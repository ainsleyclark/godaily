// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slack

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
)

func TestPlain(t *testing.T) {
	t.Parallel()
	r := Plain("hello")
	assert.Equal(t, "hello", r.Text)
	assert.Empty(t, r.Blocks.BlockSet)
	assert.Empty(t, r.Attachments)
}

func TestSuccess_WithButtons(t *testing.T) {
	t.Parallel()
	r := Success("Posted", "the body", LinkButton{
		Label: "View on Bluesky",
		URL:   "https://bsky.app/x",
		Style: "primary",
	}, LinkButton{
		Label: "View on LinkedIn",
		URL:   "https://linkedin.com/y",
	})

	assert.Equal(t, "Posted - the body", r.Text)
	assert.Len(t, r.Attachments, 1)
	assert.Equal(t, ColorSuccess, r.Attachments[0].Color)
	assert.Len(t, r.Blocks.BlockSet, 3)

	action, ok := r.Blocks.BlockSet[2].(*slack.ActionBlock)
	assert.True(t, ok, "third block should be an action block")
	assert.Len(t, action.Elements.ElementSet, 2)

	first := action.Elements.ElementSet[0].(*slack.ButtonBlockElement)
	assert.Equal(t, "View on Bluesky", first.Text.Text)
	assert.Equal(t, "https://bsky.app/x", first.URL)
	assert.Equal(t, slack.Style("primary"), first.Style)

	second := action.Elements.ElementSet[1].(*slack.ButtonBlockElement)
	assert.Equal(t, slack.Style(""), second.Style)
}

func TestError(t *testing.T) {
	t.Parallel()
	r := Error("Boom", errors.New("the cause"))
	assert.Equal(t, "Boom: the cause", r.Text)
	assert.Equal(t, ColorError, r.Attachments[0].Color)
	assert.Len(t, r.Blocks.BlockSet, 2)

	section, ok := r.Blocks.BlockSet[1].(*slack.SectionBlock)
	assert.True(t, ok, "second block should be a section block")
	assert.Equal(t, "```\nthe cause\n```", section.Text.Text)
}

func TestErrorWithContext(t *testing.T) {
	t.Parallel()
	r := ErrorWithContext("Boom", errors.New("the cause"), "`POST /x` · now")
	assert.Equal(t, "Boom: the cause", r.Text)
	assert.Len(t, r.Blocks.BlockSet, 3)

	ctx, ok := r.Blocks.BlockSet[2].(*slack.ContextBlock)
	assert.True(t, ok, "third block should be a context block")
	assert.Equal(t, "`POST /x` · now", ctx.ContextElements.Elements[0].(*slack.TextBlockObject).Text)
}

func TestError_NilErr(t *testing.T) {
	t.Parallel()
	r := Error("Boom", nil)
	assert.Equal(t, "Boom", r.Text)
	assert.Len(t, r.Blocks.BlockSet, 1)
}

func TestInfoWarn(t *testing.T) {
	t.Parallel()
	assert.Equal(t, ColorInfo, Info("t", "b").Attachments[0].Color)
	assert.Equal(t, ColorWarn, Warn("t", "b").Attachments[0].Color)
}
