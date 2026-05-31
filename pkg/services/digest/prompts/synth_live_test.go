// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prompts

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/pkg/ai/anthropic"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
)

// TestLiveSynthesise regenerates the subject + intro for one or more existing
// issues using the live prompt and model, printing the stored values next to
// the freshly generated ones so the new editorial voice can be compared
// against what actually shipped. It mutates nothing — issues are fetched
// read-only and synthesis runs locally.
//
// It is skipped unless opted into. To run:
//
//	GODAILY_SYNTH_SLUGS=2026-05-22,2026-05-15 \
//	GODAILY_API_KEY=<key> ANTHROPIC_API_KEY=<key> \
//	go test ./pkg/services/digest/prompts -run TestLiveSynthesise -v
//
// GODAILY_API_URL defaults to https://godaily.dev.
func TestLiveSynthesise(t *testing.T) {
	slugs := strings.Split(os.Getenv("GODAILY_SYNTH_SLUGS"), ",")
	apiKey := os.Getenv("GODAILY_API_KEY")
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	if os.Getenv("GODAILY_SYNTH_SLUGS") == "" || apiKey == "" || anthropicKey == "" {
		t.Skip("set GODAILY_SYNTH_SLUGS, GODAILY_API_KEY and ANTHROPIC_API_KEY to run")
	}

	base := os.Getenv("GODAILY_API_URL")
	if base == "" {
		base = "https://godaily.dev"
	}

	prompter := anthropic.New(anthropicKey)

	for _, slug := range slugs {
		slug = strings.TrimSpace(slug)
		if slug == "" {
			continue
		}
		t.Run(slug, func(t *testing.T) {
			issue := fetchIssue(t, base, apiKey, slug)
			day, err := time.Parse("2006-01-02", slug)
			require.NoError(t, err)

			meta, err := Synthesise(context.Background(), prompter, day, groupBySource(issue.Items))
			require.NoError(t, err)

			t.Logf(
				"\n=== %s (%d items) ===\n"+
					"OLD subject: %s\nNEW subject: %s\n\n"+
					"OLD intro:   %s\nNEW intro:   %s\n",
				slug, len(issue.Items),
				issue.Subject, meta.Title,
				issue.Summary, meta.Intro,
			)
		})
	}
}

type liveIssue struct {
	Subject string      `json:"subject"`
	Summary string      `json:"summary"`
	Items   []news.Item `json:"items"`
}

// fetchIssue reads a single issue (with its items) from the live API.
func fetchIssue(t *testing.T, base, apiKey, slug string) liveIssue {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, base+"/api/issues/"+slug, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode, "fetching %s", slug)

	var body struct {
		Data liveIssue `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.NotEmpty(t, body.Data.Items, "issue %s has no items", slug)
	return body.Data
}

// groupBySource buckets a flat item list into per-source SourceItems,
// mirroring what the build pipeline feeds to Synthesise. Order within a source
// is preserved; filterItems re-sorts by score, so cross-source order is
// irrelevant here.
func groupBySource(items []news.Item) []news.SourceItems {
	order := make([]news.Source, 0)
	bySource := make(map[news.Source]*news.SourceItems)
	for _, it := range items {
		if _, ok := bySource[it.Source]; !ok {
			bySource[it.Source] = &news.SourceItems{Source: it.Source}
			order = append(order, it.Source)
		}
		bySource[it.Source].Items = append(bySource[it.Source].Items, it)
	}
	out := make([]news.SourceItems, 0, len(order))
	for _, src := range order {
		out = append(out, *bySource[src])
	}
	return out
}
