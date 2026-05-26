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

	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
)

func TestAggregator_SendPreview(t *testing.T) {
	day := func(s string) time.Time {
		t.Helper()
		d, err := time.Parse("2006-01-02", s)
		require.NoError(t, err)
		return d
	}

	t.Run("Sends Digest To Admin And Leaves Status As Draft", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		date := day("2026-05-10")
		stored, err := issueRepo.Create(t.Context(), digest.Issue{
			Slug:    "2026-05-10",
			Subject: "Go 1.25 ships",
			Status:  digest.IssueStatusDraft,
			SentAt:  time.Now().UTC(),
		})
		require.NoError(t, err)

		m := &mockEmail{}
		agg := Aggregator{email: m, adminEmailAddress: "owner@example.com", issues: issueRepo, items: itemRepo}

		require.NoError(t, agg.SendPreview(t.Context(), date))

		assert.True(t, m.called)
		assert.Equal(t, []string{"owner@example.com"}, m.req.To)
		assert.Equal(t, "Go 1.25 ships", m.req.Subject)

		// Issue must remain draft so SendDigest can still send to subscribers.
		updated, err := issueRepo.Find(t.Context(), stored.ID)
		require.NoError(t, err)
		assert.Equal(t, digest.IssueStatusDraft, updated.Status)
	})

	t.Run("Returns Error When Issue Not Found", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		agg := Aggregator{email: &mockEmail{}, adminEmailAddress: "owner@example.com", issues: issueRepo, items: itemRepo}

		err := agg.SendPreview(t.Context(), day("1999-01-01"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no digest found")
	})

	t.Run("Returns Error When Status Not Draft", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		_, err := issueRepo.Create(t.Context(), digest.Issue{
			Slug:    "2026-05-11",
			Subject: "GoDaily - 2026-05-11",
			Status:  digest.IssueStatusSent,
			SentAt:  time.Now().UTC(),
		})
		require.NoError(t, err)

		agg := Aggregator{email: &mockEmail{}, adminEmailAddress: "owner@example.com", issues: issueRepo, items: itemRepo}

		sendErr := agg.SendPreview(t.Context(), day("2026-05-11"))
		require.Error(t, sendErr)
		assert.Contains(t, sendErr.Error(), "expected")
	})

	t.Run("Returns Error When Loading Sections Fails", func(t *testing.T) {
		issueRepo, _ := newTestStores(t)
		date := day("2026-05-12")
		_, err := issueRepo.Create(t.Context(), digest.Issue{
			Slug:    "2026-05-12",
			Subject: "GoDaily - 2026-05-12",
			Status:  digest.IssueStatusDraft,
			SentAt:  time.Now().UTC(),
		})
		require.NoError(t, err)

		badItems := errItemRepo{err: errors.New("db failure")}
		agg := Aggregator{email: &mockEmail{}, adminEmailAddress: "owner@example.com", issues: issueRepo, items: badItems}

		err = agg.SendPreview(t.Context(), date)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "loading sections")
	})

	t.Run("Returns Error When Rendering Fails", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		date := day("2026-05-13")
		stored, err := issueRepo.Create(t.Context(), digest.Issue{
			Slug:    "2026-05-13",
			Subject: "GoDaily - 2026-05-13",
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

		orig := htmlTmpl
		htmlTmpl = htmltemplate.Must(htmltemplate.New("digest").Parse(`{{ .Missing.NotAField }}`))
		t.Cleanup(func() { htmlTmpl = orig })

		agg := Aggregator{email: &mockEmail{}, adminEmailAddress: "owner@example.com", issues: issueRepo, items: itemRepo}
		err = agg.SendPreview(t.Context(), date)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "rendering digest")
	})

	t.Run("Returns Error When Email Send Fails", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		date := day("2026-05-14")
		_, err := issueRepo.Create(t.Context(), digest.Issue{
			Slug:    "2026-05-14",
			Subject: "GoDaily - 2026-05-14",
			Status:  digest.IssueStatusDraft,
			SentAt:  time.Now().UTC(),
		})
		require.NoError(t, err)

		m := &mockEmail{err: errors.New("smtp boom")}
		agg := Aggregator{email: m, adminEmailAddress: "owner@example.com", issues: issueRepo, items: itemRepo}

		err = agg.SendPreview(t.Context(), date)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "sending preview digest")
	})

	t.Run("Suggestion Error Does Not Fail Preview", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		date := day("2026-05-15")
		_, err := issueRepo.Create(t.Context(), digest.Issue{
			Slug:    "2026-05-15",
			Subject: "GoDaily - 2026-05-15",
			Status:  digest.IssueStatusDraft,
			SentAt:  time.Now().UTC(),
		})
		require.NoError(t, err)

		m := &mockEmail{}
		// No prompter set; SendSuggestion will return an error but SendPreview must succeed.
		agg := Aggregator{email: m, adminEmailAddress: "owner@example.com", issues: issueRepo, items: itemRepo}

		require.NoError(t, agg.SendPreview(t.Context(), date))
		assert.True(t, m.called)
	})
}
