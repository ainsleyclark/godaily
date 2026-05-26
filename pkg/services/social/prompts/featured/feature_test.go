// Copyright (c) 2026 godaily (Ainsley Clark)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package featured

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/mocks/ai"
)

func TestBuildCandidates(t *testing.T) {
	t.Parallel()

	items := []news.Item{
		{Title: "Random article", URL: "u1", Source: news.SourceMedium, Tag: news.TagArticle, Score: 0.4},
		{Title: "Go 1.30 release", URL: "u2", Source: news.SourceGoRelease, Tag: news.TagRelease, Score: 0.5},
		{Title: "Some discussion", URL: "u3", Source: news.SourceReddit, Tag: news.TagDiscussion, Score: 0.9},
		{Title: "Accepted proposal #1234", URL: "u4", Source: news.SourceGitHub, Tag: news.TagProposalAccepted, Score: 0.6},
	}

	got := buildCandidates(items)
	require.Len(t, got, 4)

	t.Run("Proposal accepted ranks above release", func(t *testing.T) {
		t.Parallel()
		// 0.6 * 0.95 = 0.57 vs 0.5 * 0.9 = 0.45
		assert.Equal(t, "u4", got[0].URL)
		assert.Equal(t, "u2", got[1].URL)
	})

	t.Run("Discussion beats article only if score is high enough", func(t *testing.T) {
		t.Parallel()
		// 0.9 * 0.5 = 0.45 vs 0.4 * 0.8 = 0.32 → discussion wins here.
		assert.Equal(t, "u3", got[2].URL)
		assert.Equal(t, "u1", got[3].URL)
	})

	t.Run("Cap respected", func(t *testing.T) {
		t.Parallel()
		many := make([]news.Item, maxCandidates+5)
		for i := range many {
			many[i] = news.Item{URL: "u", Title: "t", Tag: news.TagArticle, Score: float64(i)}
		}
		assert.Len(t, buildCandidates(many), maxCandidates)
	})
}

func TestFeature(t *testing.T) {
	t.Parallel()

	day := time.Date(2026, time.May, 20, 0, 0, 0, 0, time.UTC)
	items := []news.Item{
		{Title: "Go 1.30 released", URL: "https://go.dev/blog/go1.30", Source: news.SourceGoRelease, Tag: news.TagRelease, Score: 0.9},
		{Title: "Discussion", URL: "u2", Source: news.SourceReddit, Tag: news.TagDiscussion, Score: 0.4},
	}

	t.Run("Nil prompter errors", func(t *testing.T) {
		t.Parallel()
		_, err := Feature(t.Context(), nil, day, items)
		require.Error(t, err)
	})

	t.Run("Empty items returns ErrNoCandidates", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		p := mockai.NewMockPrompter(ctrl)
		// No prompt call expected.

		_, err := Feature(t.Context(), p, day, nil)
		assert.ErrorIs(t, err, ErrNoCandidates)
	})

	t.Run("Happy path parses Featured", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		p := mockai.NewMockPrompter(ctrl)

		resp := `{"title":"Go 1.30 released","url":"https://go.dev/blog/go1.30","source":"go_release","tag":"release","hook":"Go 1.30 ships generic type inference improvements that simplify constraints."}`
		p.EXPECT().
			Prompt(gomock.Any(), gomock.Any(), gomock.Any()).
			Return([]byte(resp), nil)

		got, err := Feature(t.Context(), p, day, items)
		require.NoError(t, err)
		assert.Equal(t, "Go 1.30 released", got.Title)
		assert.Equal(t, "https://go.dev/blog/go1.30", got.URL)
		assert.Equal(t, news.SourceGoRelease, got.Source)
		assert.Equal(t, news.TagRelease, got.Tag)
		assert.Contains(t, got.Hook, "Go 1.30")
	})

	t.Run("Strips markdown fences", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		p := mockai.NewMockPrompter(ctrl)

		fenced := "```json\n" + `{"title":"t","url":"u","source":"s","tag":"article","hook":"h"}` + "\n```"
		p.EXPECT().Prompt(gomock.Any(), gomock.Any(), gomock.Any()).Return([]byte(fenced), nil)

		got, err := Feature(t.Context(), p, day, items)
		require.NoError(t, err)
		assert.Equal(t, "u", got.URL)
	})

	t.Run("AI error wrapped", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		p := mockai.NewMockPrompter(ctrl)
		p.EXPECT().Prompt(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("boom"))

		_, err := Feature(t.Context(), p, day, items)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ai")
	})

	t.Run("Empty response errors", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		p := mockai.NewMockPrompter(ctrl)
		p.EXPECT().Prompt(gomock.Any(), gomock.Any(), gomock.Any()).Return([]byte("  "), nil)

		_, err := Feature(t.Context(), p, day, items)
		require.Error(t, err)
	})

	t.Run("Missing required fields errors", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		p := mockai.NewMockPrompter(ctrl)
		p.EXPECT().Prompt(gomock.Any(), gomock.Any(), gomock.Any()).
			Return([]byte(`{"title":"t","url":"","source":"s","tag":"article","hook":"h"}`), nil)

		_, err := Feature(context.Background(), p, day, items)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing")
	})
}

func TestParseFeatured(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		input   string
		wantErr bool
	}{
		"Happy":        {input: `{"title":"t","url":"u","source":"s","tag":"article","hook":"h"}`, wantErr: false},
		"Bad JSON":     {input: `not json`, wantErr: true},
		"Missing hook": {input: `{"title":"t","url":"u","source":"s","tag":"article"}`, wantErr: true},
		"Missing url":  {input: `{"title":"t","url":"","source":"s","tag":"article","hook":"h"}`, wantErr: true},
		"Empty body":   {input: `   `, wantErr: true},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := parseFeatured([]byte(test.input))
			assert.Equal(t, test.wantErr, err != nil)
		})
	}
}
