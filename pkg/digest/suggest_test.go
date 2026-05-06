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

package digest

import (
	"errors"
	htmltemplate "html/template"
	"testing"
	texttemplate "text/template"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/pkg/news"
	"github.com/ainsleyclark/godaily/pkg/synth"
)

var suggestDay = time.Date(2026, time.April, 26, 0, 0, 0, 0, time.UTC)

func sampleSuggestion() synth.Suggestion {
	return synth.Suggestion{
		Date: suggestDay,
		Post: "Go 1.24 is out — range-over-func is now stable.",
		References: []synth.Ref{
			{Title: "Go 1.24 Release Notes", URL: "https://go.dev/doc/go1.24", Source: news.SourceGoBlog},
		},
	}
}

func TestRenderSuggestion(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		html, text, err := renderSuggestion(sampleSuggestion())
		require.NoError(t, err)
		assert.Contains(t, html, "Go 1.24 is out")
		assert.Contains(t, html, "Go 1.24 Release Notes")
		assert.Contains(t, text, "Go 1.24 is out")
		assert.Contains(t, text, "Go 1.24 Release Notes")
	})

	// Template subtests mutate package-level vars and must run sequentially.
	t.Run("HTML Template Error", func(t *testing.T) {
		orig := suggestHTMLTmpl
		suggestHTMLTmpl = htmltemplate.Must(htmltemplate.New("suggest").Parse(`{{ .Missing.NotAField }}`))
		t.Cleanup(func() { suggestHTMLTmpl = orig })

		_, _, err := renderSuggestion(sampleSuggestion())
		assert.ErrorContains(t, err, "rendering suggest html")
	})

	t.Run("Text Template Error", func(t *testing.T) {
		orig := suggestTextTmpl
		suggestTextTmpl = texttemplate.Must(texttemplate.New("suggest").Parse(`{{ .Missing.NotAField }}`))
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

	t.Run("Sends Suggestion Email To Owner", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		date := day("2026-05-10")
		stored, err := issueRepo.Create(t.Context(), news.Issue{
			Slug:    "2026-05-10",
			Subject: "GoDaily - 2026-05-10",
			Status:  news.IssueStatusDraft,
			SentAt:  time.Now().UTC(),
		})
		require.NoError(t, err)
		_, err = itemRepo.Create(t.Context(), stored.ID, 1, news.Item{
			Source:    news.SourceDevTo,
			Title:     "item",
			URL:       "https://example.com/x",
			Published: time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour).Add(time.Hour),
		})
		require.NoError(t, err)

		m := &mockEmail{}
		sg := &mockSuggester{resp: synth.Suggestion{Post: "punchy-post"}}
		agg := Aggregator{email: m, adminEmailAddress: "to@example.com", suggester: sg, issues: issueRepo, items: itemRepo}

		require.NoError(t, agg.SendSuggestion(t.Context(), date))

		assert.True(t, sg.called)
		assert.True(t, m.called)
		assert.Contains(t, m.req.Subject, "Synth")
		assert.Contains(t, m.req.Html, "punchy-post")
		assert.Contains(t, m.req.Text, "punchy-post")
		assert.Equal(t, []string{"to@example.com"}, m.req.To)
	})

	t.Run("Returns Error When Suggester Nil", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		agg := Aggregator{email: &mockEmail{}, adminEmailAddress: "to@example.com", issues: issueRepo, items: itemRepo}

		err := agg.SendSuggestion(t.Context(), day("2026-05-11"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ANTHROPIC_API_KEY")
	})

	t.Run("Returns Error When Repos Are Nil", func(t *testing.T) {
		sg := &mockSuggester{}
		agg := Aggregator{email: &mockEmail{}, adminEmailAddress: "to@example.com", suggester: sg}

		err := agg.SendSuggestion(t.Context(), day("2026-05-12"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "persistence")
	})

	t.Run("No Items Skips Send", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		date := day("2026-05-13")
		_, err := issueRepo.Create(t.Context(), news.Issue{
			Slug:    "2026-05-13",
			Subject: "GoDaily - 2026-05-13",
			Status:  news.IssueStatusDraft,
			SentAt:  time.Now().UTC(),
		})
		require.NoError(t, err)

		m := &mockEmail{}
		sg := &mockSuggester{resp: synth.Suggestion{Post: "p"}}
		agg := Aggregator{email: m, adminEmailAddress: "to@example.com", suggester: sg, issues: issueRepo, items: itemRepo}

		require.NoError(t, agg.SendSuggestion(t.Context(), date))

		assert.False(t, sg.called)
		assert.False(t, m.called)
	})

	t.Run("No Send Address Skips Without Error", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		sg := &mockSuggester{}
		m := &mockEmail{}
		agg := Aggregator{email: m, adminEmailAddress: "", suggester: sg, issues: issueRepo, items: itemRepo}

		require.NoError(t, agg.SendSuggestion(t.Context(), day("2026-05-14")))
		assert.False(t, m.called)
	})

	t.Run("Returns Error When Issue Not Found", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		sg := &mockSuggester{}
		agg := Aggregator{email: &mockEmail{}, adminEmailAddress: "to@example.com", suggester: sg, issues: issueRepo, items: itemRepo}

		err := agg.SendSuggestion(t.Context(), day("1999-01-01"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no digest found")
	})

	t.Run("Returns Error When Loading Items Fails", func(t *testing.T) {
		issueRepo, _ := newTestStores(t)
		date := day("2026-05-17")
		_, err := issueRepo.Create(t.Context(), news.Issue{
			Slug:    "2026-05-17",
			Subject: "GoDaily - 2026-05-17",
			Status:  news.IssueStatusDraft,
			SentAt:  time.Now().UTC(),
		})
		require.NoError(t, err)

		badItems := errItemRepo{err: errors.New("db failure")}
		sg := &mockSuggester{}
		agg := Aggregator{email: &mockEmail{}, adminEmailAddress: "to@example.com", suggester: sg, issues: issueRepo, items: badItems}

		err = agg.SendSuggestion(t.Context(), date)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "loading items")
	})

	t.Run("Returns Error On Render Failure", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		date := day("2026-05-16")
		stored, err := issueRepo.Create(t.Context(), news.Issue{
			Slug:    "2026-05-16",
			Subject: "GoDaily - 2026-05-16",
			Status:  news.IssueStatusDraft,
			SentAt:  time.Now().UTC(),
		})
		require.NoError(t, err)
		_, err = itemRepo.Create(t.Context(), stored.ID, 1, news.Item{
			Source:    news.SourceDevTo,
			Title:     "item",
			URL:       "https://example.com/x",
			Published: time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour).Add(time.Hour),
		})
		require.NoError(t, err)

		orig := suggestHTMLTmpl
		suggestHTMLTmpl = htmltemplate.Must(htmltemplate.New("suggest").Parse(`{{ .Missing.NotAField }}`))
		t.Cleanup(func() { suggestHTMLTmpl = orig })

		sg := &mockSuggester{resp: synth.Suggestion{Post: "p"}}
		agg := Aggregator{email: &mockEmail{}, adminEmailAddress: "to@example.com", suggester: sg, issues: issueRepo, items: itemRepo}

		err = agg.SendSuggestion(t.Context(), date)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "rendering suggest html")
	})

	t.Run("Returns Error When Suggester Fails", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		date := day("2026-05-15")
		stored, err := issueRepo.Create(t.Context(), news.Issue{
			Slug:    "2026-05-15",
			Subject: "GoDaily - 2026-05-15",
			Status:  news.IssueStatusDraft,
			SentAt:  time.Now().UTC(),
		})
		require.NoError(t, err)
		_, err = itemRepo.Create(t.Context(), stored.ID, 1, news.Item{
			Source:    news.SourceDevTo,
			Title:     "item",
			URL:       "https://example.com/x",
			Published: time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour).Add(time.Hour),
		})
		require.NoError(t, err)

		sg := &mockSuggester{err: errors.New("anthropic down")}
		agg := Aggregator{email: &mockEmail{}, adminEmailAddress: "to@example.com", suggester: sg, issues: issueRepo, items: itemRepo}

		err = agg.SendSuggestion(t.Context(), date)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "synth")
	})
}
