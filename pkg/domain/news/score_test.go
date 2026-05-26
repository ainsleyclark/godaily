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

package news

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogScore(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		value      float64
		saturation float64
		want       float64
	}{
		"Zero value":          {value: 0, saturation: 50, want: 0},
		"At saturation":       {value: 50, saturation: 50, want: 1},
		"Above saturation":    {value: 200, saturation: 50, want: 1},
		"Negative value":      {value: -5, saturation: 50, want: 0},
		"Zero saturation":     {value: 10, saturation: 0, want: 0},
		"Negative saturation": {value: 10, saturation: -1, want: 0},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := LogScore(test.value, test.saturation)
			assert.InDelta(t, test.want, got, 1e-9)
		})
	}
}

func TestLogScore_Monotonicity(t *testing.T) {
	t.Parallel()

	prev := LogScore(1, 100)
	for v := 2.0; v < 100; v++ {
		cur := LogScore(v, 100)
		assert.GreaterOrEqual(t, cur, prev, "LogScore must be monotonically non-decreasing in value")
		prev = cur
	}
}

func TestLogScore_BoundedZeroToOne(t *testing.T) {
	t.Parallel()

	for _, v := range []float64{0, 1, 10, 50, 100, 1_000, 1_000_000} {
		got := LogScore(v, 50)
		assert.GreaterOrEqual(t, got, 0.0)
		assert.LessOrEqual(t, got, 1.0)
	}
}

func TestSourceWeight(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		source Source
		tag    Tag
		want   float64
	}{
		"Go Blog":                  {source: SourceGoBlog, tag: TagArticle, want: 2.0},
		"GitHub Proposal Accepted": {source: SourceGitHub, tag: TagProposalAccepted, want: 1.8},
		"GitHub Proposal Shipped":  {source: SourceGitHub, tag: TagProposalShipped, want: 1.7},
		"GitHub Proposal Open":     {source: SourceGitHub, tag: TagProposal, want: 1.5},
		"Hacker News":              {source: SourceHN, tag: TagArticle, want: 1.2},
		"Reddit":                   {source: SourceReddit, tag: TagArticle, want: 1.0},
		"Lobsters":                 {source: SourceLobsters, tag: TagArticle, want: 1.0},
		"Dev.to":                   {source: SourceDevTo, tag: TagArticle, want: 1.0},
		"GolangBridge":             {source: SourceGolangBridge, tag: TagArticle, want: 1.0},
		"YouTube":                  {source: SourceYouTube, tag: TagVideo, want: 1.0},
		"Medium":                   {source: SourceMedium, tag: TagArticle, want: 0.5},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := SourceWeight(test.source, test.tag)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestSourceWeight_AllSourcesCovered(t *testing.T) {
	t.Parallel()

	for _, s := range Sources {
		got := SourceWeight(s, TagArticle)
		assert.Greater(t, got, 0.0, "source %q must have a non-zero weight", s)
	}
}

func TestScoreOf(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		source    Source
		tag       Tag
		signal    float64
		hasSignal bool
		wantMin   float64
		wantMax   float64
	}{
		"HN at saturation":                  {SourceHN, TagArticle, 50, true, 1.2, 1.2},
		"HN above saturation clamped":       {SourceHN, TagArticle, 5000, true, 1.2, 1.2},
		"HN zero engagement uses floor":     {SourceHN, TagArticle, 0, true, 0.12, 0.12},
		"Reddit mid-range":                  {SourceReddit, TagArticle, 50, true, 0.7, 0.95},
		"Lobsters at saturation":            {SourceLobsters, TagArticle, 50, true, 1.0, 1.0},
		"GitHub Accepted floor carries":     {SourceGitHub, TagProposalAccepted, 0, true, 0.9, 0.9},
		"GitHub Shipped floor carries":      {SourceGitHub, TagProposalShipped, 0, true, 0.85, 0.85},
		"GitHub Proposal floor carries":     {SourceGitHub, TagProposal, 0, true, 0.75, 0.75},
		"Go Blog no signal":                 {SourceGoBlog, TagArticle, 0, false, 1.0, 1.0},
		"Medium no signal":                  {SourceMedium, TagArticle, 0, false, 0.25, 0.25},
		"YouTube no signal":                 {SourceYouTube, TagVideo, 0, false, 0.5, 0.5},
		"Dev.to zero engagement uses floor": {SourceDevTo, TagArticle, 0, true, 0.1, 0.1},
		"GolangBridge mid-range views":      {SourceGolangBridge, TagArticle, 1000, true, 0.7, 0.95},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := ScoreOf(test.source, test.tag, test.signal, test.hasSignal)
			assert.GreaterOrEqual(t, got, test.wantMin-1e-9, "score %v below expected minimum %v", got, test.wantMin)
			assert.LessOrEqual(t, got, test.wantMax+1e-9, "score %v above expected maximum %v", got, test.wantMax)
		})
	}
}

func TestScoreOf_OfficialBeatsNoisy(t *testing.T) {
	t.Parallel()

	goBlog := ScoreOf(SourceGoBlog, TagArticle, 0, false)
	medium := ScoreOf(SourceMedium, TagArticle, 0, false)
	assert.Greater(t, goBlog, medium)

	redditLow := ScoreOf(SourceReddit, TagArticle, 5, true)
	assert.Greater(t, goBlog, redditLow)

	ghAccepted := ScoreOf(SourceGitHub, TagProposalAccepted, 0, true)
	redditTypical := ScoreOf(SourceReddit, TagArticle, 50, true)
	assert.Greater(t, ghAccepted, redditTypical)
}
