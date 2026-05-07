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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/pkg/news"
	"github.com/ainsleyclark/godaily/pkg/synth"
)

func TestAggregator_SendDigest(t *testing.T) {
	day := func(s string) time.Time {
		t.Helper()
		d, err := time.Parse("2006-01-02", s)
		require.NoError(t, err)
		return d
	}

	t.Run("Sends Email And Updates Status To Sent", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		date := day("2026-04-26")
		stored, err := issueRepo.Create(t.Context(), news.Issue{
			Slug:    "2026-04-26",
			Subject: "GoDaily - 2026-04-26",
			Status:  news.IssueStatusDraft,
			SentAt:  time.Now().UTC(),
		})
		require.NoError(t, err)

		m := &mockEmail{}
		agg := Aggregator{email: m, adminEmailAddress: "to@example.com", issues: issueRepo, items: itemRepo}

		require.NoError(t, agg.SendDigest(t.Context(), date, false))

		assert.True(t, m.called)
		assert.Contains(t, m.req.Subject, "April 26, 2026")

		updated, err := issueRepo.Find(t.Context(), stored.ID)
		require.NoError(t, err)
		assert.Equal(t, news.IssueStatusSent, updated.Status)
	})

	t.Run("Email Error Updates Status To Error", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		date := day("2026-04-27")
		stored, err := issueRepo.Create(t.Context(), news.Issue{
			Slug:    "2026-04-27",
			Subject: "GoDaily - 2026-04-27",
			Status:  news.IssueStatusDraft,
			SentAt:  time.Now().UTC(),
		})
		require.NoError(t, err)

		m := &mockEmail{err: errors.New("send boom")}
		agg := Aggregator{email: m, adminEmailAddress: "to@example.com", issues: issueRepo, items: itemRepo}

		require.NoError(t, agg.SendDigest(t.Context(), date, false))

		updated, err := issueRepo.Find(t.Context(), stored.ID)
		require.NoError(t, err)
		assert.Equal(t, news.IssueStatusError, updated.Status)
	})

	t.Run("Returns Error When Issue Not Found", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		agg := Aggregator{email: &mockEmail{}, adminEmailAddress: "to@example.com", issues: issueRepo, items: itemRepo}

		err := agg.SendDigest(t.Context(), day("1999-01-01"), false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no digest found")
	})

	t.Run("Returns Error When Status Not Draft", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		_, err := issueRepo.Create(t.Context(), news.Issue{
			Slug:    "2026-04-30",
			Subject: "GoDaily - 2026-04-30",
			Status:  news.IssueStatusSent,
			SentAt:  time.Now().UTC(),
		})
		require.NoError(t, err)

		agg := Aggregator{email: &mockEmail{}, adminEmailAddress: "to@example.com", issues: issueRepo, items: itemRepo}
		sendErr := agg.SendDigest(t.Context(), day("2026-04-30"), false)
		require.Error(t, sendErr)
		assert.Contains(t, sendErr.Error(), "expected")
	})

	t.Run("Force Skips Status Check", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		date := day("2026-04-30")
		_, err := issueRepo.Create(t.Context(), news.Issue{
			Slug:    "2026-04-30",
			Subject: "GoDaily - 2026-04-30",
			Status:  news.IssueStatusSent,
			SentAt:  time.Now().UTC(),
		})
		require.NoError(t, err)

		m := &mockEmail{}
		agg := Aggregator{email: m, adminEmailAddress: "to@example.com", issues: issueRepo, items: itemRepo}

		require.NoError(t, agg.SendDigest(t.Context(), date, true))
		assert.True(t, m.called)
	})

	t.Run("Returns Error When Loading Sections Fails", func(t *testing.T) {
		issueRepo, _ := newTestStores(t)
		date := day("2026-05-02")
		_, err := issueRepo.Create(t.Context(), news.Issue{
			Slug:    "2026-05-02",
			Subject: "GoDaily - 2026-05-02",
			Status:  news.IssueStatusDraft,
			SentAt:  time.Now().UTC(),
		})
		require.NoError(t, err)

		badItems := errItemRepo{err: errors.New("db failure")}
		agg := Aggregator{email: &mockEmail{}, adminEmailAddress: "to@example.com", issues: issueRepo, items: badItems}

		err = agg.SendDigest(t.Context(), date, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "loading sections")
	})

	t.Run("Returns Error When Rendering Fails", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		date := day("2026-05-03")
		stored, err := issueRepo.Create(t.Context(), news.Issue{
			Slug:    "2026-05-03",
			Subject: "GoDaily - 2026-05-03",
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

		orig := htmlTmpl
		htmlTmpl = htmltemplate.Must(htmltemplate.New("digest").Parse(`{{ .Missing.NotAField }}`))
		t.Cleanup(func() { htmlTmpl = orig })

		agg := Aggregator{email: &mockEmail{}, adminEmailAddress: "to@example.com", issues: issueRepo, items: itemRepo}
		err = agg.SendDigest(t.Context(), date, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "rendering digest")
	})

	t.Run("Synth Never Called During Send", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		date := day("2026-05-01")
		stored, err := issueRepo.Create(t.Context(), news.Issue{
			Slug:    "2026-05-01",
			Subject: "GoDaily - 2026-05-01",
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

		require.NoError(t, agg.SendDigest(t.Context(), date, false))

		assert.False(t, sg.called, "synth must not be called during Send")
		assert.True(t, m.called)
		assert.NotContains(t, m.req.Html, "punchy-post")
	})
}

func TestLoadSections(t *testing.T) {
	t.Run("Empty Issue Returns Empty Sections", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		issue, err := issueRepo.Create(t.Context(), news.Issue{
			Slug:   "2026-01-01",
			Status: news.IssueStatusDraft,
			SentAt: time.Now().UTC(),
		})
		require.NoError(t, err)

		sections, err := loadSections(t.Context(), itemRepo, issue.ID)
		require.NoError(t, err)
		assert.Empty(t, sections)
	})

	t.Run("Groups Items By Source And Sorts By Priority", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		issue, err := issueRepo.Create(t.Context(), news.Issue{
			Slug:   "2026-01-02",
			Status: news.IssueStatusDraft,
			SentAt: time.Now().UTC(),
		})
		require.NoError(t, err)

		published := time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour).Add(time.Hour)

		// Insert two sources — GoBlog has higher priority than Medium.
		_, err = itemRepo.Create(t.Context(), issue.ID, 1, news.Item{Source: news.SourceMedium, Title: "medium-1", URL: "https://medium.com/1", Published: published})
		require.NoError(t, err)
		_, err = itemRepo.Create(t.Context(), issue.ID, 2, news.Item{Source: news.SourceGoBlog, Title: "goblog-1", URL: "https://go.dev/1", Published: published})
		require.NoError(t, err)
		_, err = itemRepo.Create(t.Context(), issue.ID, 3, news.Item{Source: news.SourceGoBlog, Title: "goblog-2", URL: "https://go.dev/2", Published: published})
		require.NoError(t, err)

		sections, err := loadSections(t.Context(), itemRepo, issue.ID)
		require.NoError(t, err)
		require.Len(t, sections, 2)

		// GoBlog has higher priority so comes first.
		assert.Equal(t, news.SourceGoBlog, sections[0].Source)
		assert.Len(t, sections[0].Items, 2)
		assert.Equal(t, news.SourceMedium, sections[1].Source)
		assert.Len(t, sections[1].Items, 1)
	})
}
