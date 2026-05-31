// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"encoding/json"
	"errors"
	htmltemplate "html/template"
	"testing"
	texttemplate "text/template"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	digest "github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/mocks/ai"
	"github.com/ainsleyclark/godaily/pkg/services/digest/prompts"
)

var suggestDay = time.Date(2026, time.April, 26, 0, 0, 0, 0, time.UTC)

func sampleSuggestion() prompts.Suggestion {
	return prompts.Suggestion{
		Date: suggestDay,
		Posts: []prompts.Post{
			{
				Text: "Go 1.24 is out — range-over-func is now stable.",
				References: []prompts.Ref{
					{Title: "Go 1.24 Release Notes", URL: "https://go.dev/doc/go1.24", Source: news.SourceGoBlog},
				},
			},
			{
				Text: "A new proposal lands for structured logging in the standard library.",
				References: []prompts.Ref{
					{Title: "slog proposal", URL: "https://go.dev/issue/56345", Source: news.SourceHN},
				},
			},
			{
				Text: "Sharp write-up on profiling allocation hot paths in production Go services.",
				References: []prompts.Ref{
					{Title: "Profiling Go allocations", URL: "https://example.com/profiling", Source: news.SourceDevTo},
				},
			},
		},
	}
}

func TestRenderSuggestion(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		html, text, err := renderSuggestion(sampleSuggestion())
		require.NoError(t, err)
		assert.Contains(t, html, "Go 1.24 is out")
		assert.Contains(t, html, "Go 1.24 Release Notes")
		assert.Contains(t, html, "Post 3")
		assert.Contains(t, html, "Profiling Go allocations")
		assert.Contains(t, html, "Sent by")    // shared layout footer
		assert.NotContains(t, html, "&mdash;") // no em-dash in the header
		assert.Contains(t, text, "Go 1.24 is out")
		assert.Contains(t, text, "Go 1.24 Release Notes")
		assert.Contains(t, text, "Post 3")
	})

	// Template subtests mutate package-level vars and must run sequentially.
	t.Run("HTML Template Error", func(t *testing.T) {
		orig := suggestHTMLTmpl
		suggestHTMLTmpl = htmltemplate.Must(htmltemplate.New("suggest-html").Parse(`{{ define "email-layout" }}{{ .Missing.NotAField }}{{ end }}`))
		t.Cleanup(func() { suggestHTMLTmpl = orig })

		_, _, err := renderSuggestion(sampleSuggestion())
		assert.ErrorContains(t, err, "rendering suggest html")
	})

	t.Run("Text Template Error", func(t *testing.T) {
		orig := suggestTextTmpl
		suggestTextTmpl = texttemplate.Must(texttemplate.New("suggest-text").Parse(`{{ define "email-layout-text" }}{{ .Missing.NotAField }}{{ end }}`))
		t.Cleanup(func() { suggestTextTmpl = orig })

		_, _, err := renderSuggestion(sampleSuggestion())
		assert.ErrorContains(t, err, "rendering suggest text")
	})
}

func TestAggregator_SendSuggestion(t *testing.T) {
	day := func(s string) time.Time {
		t.Helper()
		d, err := time.Parse("2006-01-02", s)
		require.NoError(t, err)
		return d
	}

	validJSON := func(posts ...string) []byte {
		arr := make([]map[string]any, len(posts))
		for i, post := range posts {
			arr[i] = map[string]any{"post": post, "references": []any{}}
		}
		raw, _ := json.Marshal(map[string]any{"posts": arr})
		return raw
	}

	t.Run("Sends Suggestion Email To Owner", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		date := day("2026-05-10")
		stored, err := issueRepo.Create(t.Context(), digest.Issue{
			Slug:    "2026-05-10",
			Subject: "GoDaily - 2026-05-10",
			Status:  digest.IssueStatusDraft,
			SentAt:  time.Now().UTC(),
		})
		require.NoError(t, err)
		_, err = itemRepo.Create(t.Context(), &stored.ID, 1, news.Item{
			Source:    news.SourceDevTo,
			Title:     "item",
			URL:       "https://example.com/x",
			Published: time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour).Add(time.Hour),
		})
		require.NoError(t, err)

		m := &mockEmail{}
		p := mockai.NewMockPrompter(gomock.NewController(t))
		p.EXPECT().Prompt(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(validJSON("first-post", "second-post", "third-post"), nil)
		agg := Service{email: m, adminEmailAddress: "to@example.com", prompter: p, issues: issueRepo, items: itemRepo}

		require.NoError(t, agg.SendSuggestion(t.Context(), date))

		assert.True(t, m.called)
		assert.Contains(t, m.req.Subject, "Synth")
		assert.Contains(t, m.req.Html, "first-post")
		assert.Contains(t, m.req.Html, "third-post")
		assert.Contains(t, m.req.Text, "first-post")
		assert.Equal(t, []string{"to@example.com"}, m.req.To)
	})

	t.Run("Returns Error When Prompter Nil", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		agg := Service{email: &mockEmail{}, adminEmailAddress: "to@example.com", issues: issueRepo, items: itemRepo}

		err := agg.SendSuggestion(t.Context(), day("2026-05-11"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ANTHROPIC_API_KEY")
	})

	t.Run("Returns Error When Repos Are Nil", func(t *testing.T) {
		p := mockai.NewMockPrompter(gomock.NewController(t))
		agg := Service{email: &mockEmail{}, adminEmailAddress: "to@example.com", prompter: p}

		err := agg.SendSuggestion(t.Context(), day("2026-05-12"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "persistence")
	})

	t.Run("No Items Skips Send", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		date := day("2026-05-13")
		_, err := issueRepo.Create(t.Context(), digest.Issue{
			Slug:    "2026-05-13",
			Subject: "GoDaily - 2026-05-13",
			Status:  digest.IssueStatusDraft,
			SentAt:  time.Now().UTC(),
		})
		require.NoError(t, err)

		m := &mockEmail{}
		p := mockai.NewMockPrompter(gomock.NewController(t))
		agg := Service{email: m, adminEmailAddress: "to@example.com", prompter: p, issues: issueRepo, items: itemRepo}

		require.NoError(t, agg.SendSuggestion(t.Context(), date))

		assert.False(t, m.called)
	})

	t.Run("No Send Address Skips Without Error", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		p := mockai.NewMockPrompter(gomock.NewController(t))
		m := &mockEmail{}
		agg := Service{email: m, adminEmailAddress: "", prompter: p, issues: issueRepo, items: itemRepo}

		require.NoError(t, agg.SendSuggestion(t.Context(), day("2026-05-14")))
		assert.False(t, m.called)
	})

	t.Run("Returns Error When Issue Not Found", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		p := mockai.NewMockPrompter(gomock.NewController(t))
		agg := Service{email: &mockEmail{}, adminEmailAddress: "to@example.com", prompter: p, issues: issueRepo, items: itemRepo}

		err := agg.SendSuggestion(t.Context(), day("1999-01-01"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no digest found")
	})

	t.Run("Returns Error When Loading Items Fails", func(t *testing.T) {
		issueRepo, _ := newTestStores(t)
		date := day("2026-05-17")
		_, err := issueRepo.Create(t.Context(), digest.Issue{
			Slug:    "2026-05-17",
			Subject: "GoDaily - 2026-05-17",
			Status:  digest.IssueStatusDraft,
			SentAt:  time.Now().UTC(),
		})
		require.NoError(t, err)

		badItems := errItemRepo{err: errors.New("db failure")}
		p := mockai.NewMockPrompter(gomock.NewController(t))
		agg := Service{email: &mockEmail{}, adminEmailAddress: "to@example.com", prompter: p, issues: issueRepo, items: badItems}

		err = agg.SendSuggestion(t.Context(), date)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "loading items")
	})

	t.Run("Returns Error On Render Failure", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		date := day("2026-05-16")
		stored, err := issueRepo.Create(t.Context(), digest.Issue{
			Slug:    "2026-05-16",
			Subject: "GoDaily - 2026-05-16",
			Status:  digest.IssueStatusDraft,
			SentAt:  time.Now().UTC(),
		})
		require.NoError(t, err)
		_, err = itemRepo.Create(t.Context(), &stored.ID, 1, news.Item{
			Source:    news.SourceDevTo,
			Title:     "item",
			URL:       "https://example.com/x",
			Published: time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour).Add(time.Hour),
		})
		require.NoError(t, err)

		orig := suggestHTMLTmpl
		suggestHTMLTmpl = htmltemplate.Must(htmltemplate.New("suggest-html").Parse(`{{ define "email-layout" }}{{ .Missing.NotAField }}{{ end }}`))
		t.Cleanup(func() { suggestHTMLTmpl = orig })

		p := mockai.NewMockPrompter(gomock.NewController(t))
		p.EXPECT().Prompt(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(validJSON("p"), nil)
		agg := Service{email: &mockEmail{}, adminEmailAddress: "to@example.com", prompter: p, issues: issueRepo, items: itemRepo}

		err = agg.SendSuggestion(t.Context(), date)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "rendering suggest html")
	})

	t.Run("Returns Error When Prompter Fails", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		date := day("2026-05-15")
		stored, err := issueRepo.Create(t.Context(), digest.Issue{
			Slug:    "2026-05-15",
			Subject: "GoDaily - 2026-05-15",
			Status:  digest.IssueStatusDraft,
			SentAt:  time.Now().UTC(),
		})
		require.NoError(t, err)
		_, err = itemRepo.Create(t.Context(), &stored.ID, 1, news.Item{
			Source:    news.SourceDevTo,
			Title:     "item",
			URL:       "https://example.com/x",
			Published: time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour).Add(time.Hour),
		})
		require.NoError(t, err)

		p := mockai.NewMockPrompter(gomock.NewController(t))
		p.EXPECT().Prompt(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("anthropic down"))
		agg := Service{email: &mockEmail{}, adminEmailAddress: "to@example.com", prompter: p, issues: issueRepo, items: itemRepo}

		err = agg.SendSuggestion(t.Context(), date)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "synth")
	})

	t.Run("Prompter Error Sends Slack Notification", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		date := day("2026-05-18")
		stored, err := issueRepo.Create(t.Context(), digest.Issue{
			Slug:    "2026-05-18",
			Subject: "GoDaily - 2026-05-18",
			Status:  digest.IssueStatusDraft,
			SentAt:  time.Now().UTC(),
		})
		require.NoError(t, err)
		_, err = itemRepo.Create(t.Context(), &stored.ID, 1, news.Item{
			Source:    news.SourceDevTo,
			Title:     "item",
			URL:       "https://example.com/x",
			Published: time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour).Add(time.Hour),
		})
		require.NoError(t, err)

		p := mockai.NewMockPrompter(gomock.NewController(t))
		p.EXPECT().Prompt(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("rate limited"))
		sl := &mockSlack{}
		agg := Service{email: &mockEmail{}, adminEmailAddress: "to@example.com", prompter: p, slack: sl, issues: issueRepo, items: itemRepo}

		err = agg.SendSuggestion(t.Context(), date)
		require.Error(t, err)
		require.Len(t, sl.msgs, 1)
		assert.Contains(t, sl.msgs[0], "AI suggestion failed")
		assert.Contains(t, sl.msgs[0], "rate limited")
	})
}
