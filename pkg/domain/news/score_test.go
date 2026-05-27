// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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
		"HN Who's Hiring":          {source: SourceHNJobs, tag: TagJobs, want: 1.4},
		"Remote OK":                {source: SourceRemoteOK, tag: TagJobs, want: 1.0},
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

func TestJobBoost(t *testing.T) {
	t.Parallel()

	t.Run("Fresh listing with everything", func(t *testing.T) {
		t.Parallel()
		// goBoost(1.5) * salaryBoost(1.2) * remoteBoost(1.15) * recency(exp(0)=1) ≈ 2.07
		got := JobBoost(0, true, true, true)
		assert.InDelta(t, 1.5*1.2*1.15, got, 1e-9)
	})

	t.Run("Fresh bare listing", func(t *testing.T) {
		t.Parallel()
		// All flags false, age 0: boost is 1.0.
		assert.InDelta(t, 1.0, JobBoost(0, false, false, false), 1e-9)
	})

	t.Run("Age decays score", func(t *testing.T) {
		t.Parallel()
		fresh := JobBoost(0, true, true, true)
		week := JobBoost(7, true, true, true)
		stale := JobBoost(28, true, true, true)
		assert.Greater(t, fresh, week)
		assert.Greater(t, week, stale)
	})

	t.Run("Recency floors out far past", func(t *testing.T) {
		t.Parallel()
		// At very large ages the recency factor clamps at 0.1, so the boost
		// stops collapsing to zero. With all flags on this is 0.1 * 2.07.
		got := JobBoost(365, true, true, true)
		assert.InDelta(t, 0.1*1.5*1.2*1.15, got, 1e-9)
	})

	t.Run("Negative age treated as zero", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, JobBoost(0, false, false, false), JobBoost(-3, false, false, false))
	})

	t.Run("Go-in-title boost outweighs salary alone", func(t *testing.T) {
		t.Parallel()
		// 1.5 (Go) > 1.2 (salary), so a Go-titled bare listing should beat
		// a non-Go listing that discloses salary.
		assert.Greater(t, JobBoost(0, true, false, false), JobBoost(0, false, true, false))
	})
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
