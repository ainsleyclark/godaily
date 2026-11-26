// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rotation

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/social"
	mockai "github.com/ainsleyclark/godaily/pkg/mocks/ai"
	"github.com/ainsleyclark/godaily/pkg/services/social/compose"
)

// w46Items mirrors the shape of the W46 recap that was cut off at
// "Building a multi-agent code review system in Go with planner,…" on
// Bluesky: long titles and long URLs.
var w46Items = []RecapItem{
	{Title: "Building a multi-agent code review system in Go with planner, reviewer and fixer agents", URL: "https://dev.to/someone/building-a-multi-agent-code-review-system-in-go-with-planner-reviewer-and-fixer-agents-4k2j", Source: "devto", Clicks: 40},
	{Title: "Go 1.26 release notes: what changed in the runtime and toolchain", URL: "https://go.dev/doc/go1.26", Source: "go_blog", Clicks: 30},
	{Title: "Understanding escape analysis in the Go compiler with practical examples", URL: "https://medium.com/@someone/understanding-escape-analysis-in-the-go-compiler-with-practical-examples-1a2b3c4d5e6f", Source: "medium", Clicks: 20},
}

func recapResponse(t *testing.T, intro string, phrases ...string) []byte {
	t.Helper()
	b, err := json.Marshal(map[string]any{"intro": intro, "phrases": phrases})
	require.NoError(t, err)
	return b
}

func TestRecap(t *testing.T) {
	t.Parallel()

	payload := RecapPayload{WeekLabel: "2026-W46", Items: w46Items}

	for _, plat := range []social.Platform{social.Bluesky, social.LinkedIn, social.Mastodon} {
		t.Run(plat.String(), func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			p := mockai.NewMockPrompter(ctrl)
			p.EXPECT().Prompt(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(recapResponse(t, "Most-clicked Go stories last week:",
					"A multi-agent code review system built in Go",
					"What changed in the Go 1.26 runtime and toolchain",
					"How the Go compiler decides what escapes to the heap",
				), nil)

			got, err := Recap(t.Context(), p, plat, payload)
			require.NoError(t, err)

			assert.LessOrEqual(t, compose.Length(plat, got), compose.Limit(plat), "recap must fit the platform limit")
			assert.True(t, strings.HasSuffix(got, compose.SubscribeLine(plat, "recap")), "recap must end with the Subscribe line")
			assert.NotContains(t, got, "…", "recap must never be cut off mid-sentence")
			assert.Contains(t, got, w46Items[0].URL, "top item is always kept, URL verbatim")
			for _, tag := range platformProfiles[plat].hashtags {
				assert.Contains(t, got, tag)
			}
		})
	}

	t.Run("Nil prompter errors", func(t *testing.T) {
		t.Parallel()
		_, err := Recap(t.Context(), nil, social.Bluesky, payload)
		require.Error(t, err)
	})

	t.Run("No items errors", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		_, err := Recap(t.Context(), mockai.NewMockPrompter(ctrl), social.Bluesky, RecapPayload{})
		require.Error(t, err)
	})

	t.Run("Bad JSON errors", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		p := mockai.NewMockPrompter(ctrl)
		p.EXPECT().Prompt(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]byte("nope"), nil)
		_, err := Recap(t.Context(), p, social.Bluesky, payload)
		require.Error(t, err)
	})
}

func TestAssembleRecap(t *testing.T) {
	t.Parallel()

	items := []RecapItem{
		{Title: "Title A", URL: "https://a.dev/1"},
		{Title: "Title B", URL: "https://b.dev/2"},
		{Title: "Title C", URL: "https://c.dev/3"},
	}

	t.Run("Lays out intro, entries, hashtags and footer", func(t *testing.T) {
		t.Parallel()
		got := assembleRecap(social.LinkedIn, []string{"#golang"}, "Top Go last week:", []string{"Phrase A", "Phrase B", "Phrase C"}, items)
		want := "Top Go last week:\n\n" +
			"Phrase A\nhttps://a.dev/1\n\n" +
			"Phrase B\nhttps://b.dev/2\n\n" +
			"Phrase C\nhttps://c.dev/3\n\n" +
			"#golang\n\n" +
			compose.SubscribeLine(social.LinkedIn, "recap")
		assert.Equal(t, want, got)
	})

	t.Run("Missing phrases and intro fall back", func(t *testing.T) {
		t.Parallel()
		got := assembleRecap(social.LinkedIn, nil, "", []string{"Phrase A", "  "}, items)
		assert.True(t, strings.HasPrefix(got, defaultRecapIntro+"\n\n"))
		assert.Contains(t, got, "Title B\nhttps://b.dev/2")
		assert.Contains(t, got, "Title C\nhttps://c.dev/3")
	})

	t.Run("Phrases with newlines are flattened", func(t *testing.T) {
		t.Parallel()
		got := assembleRecap(social.LinkedIn, nil, "Intro\nline", []string{"one\ntwo", "B", "C"}, items)
		assert.True(t, strings.HasPrefix(got, "Intro line\n\none two\nhttps://a.dev/1"))
	})

	t.Run("Drops lowest-ranked items to fit rather than truncating", func(t *testing.T) {
		t.Parallel()
		long := strings.Repeat("x", 120)
		got := assembleRecap(social.Bluesky, []string{"#golang"}, "Intro:", []string{long, long, long}, items)
		assert.LessOrEqual(t, compose.Length(social.Bluesky, got), compose.Limit(social.Bluesky))
		assert.Contains(t, got, "https://a.dev/1")
		assert.NotContains(t, got, "https://c.dev/3")
		assert.True(t, strings.HasSuffix(got, compose.SubscribeLine(social.Bluesky, "recap")))
	})

	t.Run("Footer survives even when one item cannot fit", func(t *testing.T) {
		t.Parallel()
		huge := strings.Repeat("word ", 100)
		got := assembleRecap(social.Bluesky, []string{"#golang"}, "Intro:", []string{huge}, items[:1])
		assert.LessOrEqual(t, compose.Length(social.Bluesky, got), compose.Limit(social.Bluesky))
		assert.True(t, strings.HasSuffix(got, compose.SubscribeLine(social.Bluesky, "recap")))
	})
}
