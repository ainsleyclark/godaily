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
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	domainsubscriber "github.com/ainsleyclark/godaily/pkg/domain/subscriber"
	"github.com/ainsleyclark/godaily/pkg/gateway/email"
	mockai "github.com/ainsleyclark/godaily/pkg/mocks/ai"
	mocksubscriber "github.com/ainsleyclark/godaily/pkg/mocks/subscriber"
)

func TestAggregator_SendDigest(t *testing.T) {
	day := func(s string) time.Time {
		t.Helper()
		d, err := time.Parse("2006-01-02", s)
		require.NoError(t, err)
		return d
	}

	t.Run("Sends Email To Subscribers And Updates Status To Sent", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		date := day("2026-04-26")
		stored, err := issueRepo.Create(t.Context(), news.Issue{
			Slug:    "2026-04-26",
			Subject: "Go 1.24 lands — goroutines got faster",
			Status:  news.IssueStatusDraft,
			SentAt:  time.Now().UTC(),
		})
		require.NoError(t, err)

		subs := mocksubscriber.NewMockSubscriberRepository(gomock.NewController(t))
		subs.EXPECT().ListActive(gomock.Any()).Return([]domainsubscriber.Subscriber{
			{ID: 1, Email: "reader@example.com", UnsubscribeToken: "tok-1"},
		}, nil)

		m := &mockEmail{}
		agg := Aggregator{email: m, issues: issueRepo, items: itemRepo, subscribers: subs}

		require.NoError(t, agg.SendDigest(t.Context(), date, false))

		assert.True(t, m.called)
		assert.Equal(t, "Go 1.24 lands — goroutines got faster", m.req.Subject)

		updated, err := issueRepo.Find(t.Context(), stored.ID)
		require.NoError(t, err)
		assert.Equal(t, news.IssueStatusSent, updated.Status)
	})

	t.Run("Subscriber Email Error Still Updates Status To Sent", func(t *testing.T) {
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
		agg := Aggregator{email: m, issues: issueRepo, items: itemRepo, subscribers: newSubsMock(t)}

		require.NoError(t, agg.SendDigest(t.Context(), date, false))

		updated, err := issueRepo.Find(t.Context(), stored.ID)
		require.NoError(t, err)
		assert.Equal(t, news.IssueStatusSent, updated.Status)
	})

	t.Run("Returns Error When Issue Not Found", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		agg := Aggregator{email: &mockEmail{}, adminEmailAddress: "to@example.com", issues: issueRepo, items: itemRepo, subscribers: newSubsMock(t)}

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

		agg := Aggregator{email: &mockEmail{}, adminEmailAddress: "to@example.com", issues: issueRepo, items: itemRepo, subscribers: newSubsMock(t)}
		sendErr := agg.SendDigest(t.Context(), day("2026-04-30"), false)
		require.Error(t, sendErr)
		assert.Contains(t, sendErr.Error(), "expected")
	})

	t.Run("Force Skips Status Check", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		date := day("2026-04-30")
		stored, err := issueRepo.Create(t.Context(), news.Issue{
			Slug:    "2026-04-30",
			Subject: "GoDaily - 2026-04-30",
			Status:  news.IssueStatusSent,
			SentAt:  time.Now().UTC(),
		})
		require.NoError(t, err)

		agg := Aggregator{email: &mockEmail{}, issues: issueRepo, items: itemRepo, subscribers: newSubsMock(t)}

		require.NoError(t, agg.SendDigest(t.Context(), date, true))

		updated, err := issueRepo.Find(t.Context(), stored.ID)
		require.NoError(t, err)
		assert.Equal(t, news.IssueStatusSent, updated.Status)
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
		agg := Aggregator{email: &mockEmail{}, adminEmailAddress: "to@example.com", issues: issueRepo, items: badItems, subscribers: newSubsMock(t)}

		err = agg.SendDigest(t.Context(), date, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "loading sections")
	})

	t.Run("Subscriber Render Error Is Skipped", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		date := day("2026-05-03")
		stored, err := issueRepo.Create(t.Context(), news.Issue{
			Slug:    "2026-05-03",
			Subject: "GoDaily - 2026-05-03",
			Status:  news.IssueStatusDraft,
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

		subs := mocksubscriber.NewMockSubscriberRepository(gomock.NewController(t))
		subs.EXPECT().ListActive(gomock.Any()).Return([]domainsubscriber.Subscriber{
			{ID: 1, Email: "reader@example.com", UnsubscribeToken: "tok-1"},
		}, nil)

		m := &mockEmail{}
		agg := Aggregator{email: m, issues: issueRepo, items: itemRepo, subscribers: subs}

		// Subscriber rendering errors are non-fatal; the digest is still marked sent.
		require.NoError(t, agg.SendDigest(t.Context(), date, false))
		assert.False(t, m.called)

		updated, err := issueRepo.Find(t.Context(), stored.ID)
		require.NoError(t, err)
		assert.Equal(t, news.IssueStatusSent, updated.Status)
	})

	t.Run("Tags Subscriber Email With Issue And Subscriber IDs", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		date := day("2026-05-10")
		stored, err := issueRepo.Create(t.Context(), news.Issue{
			Slug:    "2026-05-10",
			Subject: "GoDaily - 2026-05-10",
			Status:  news.IssueStatusDraft,
			SentAt:  time.Now().UTC(),
		})
		require.NoError(t, err)

		subs := mocksubscriber.NewMockSubscriberRepository(gomock.NewController(t))
		subs.EXPECT().ListActive(gomock.Any()).Return([]domainsubscriber.Subscriber{
			{ID: 99, Email: "reader@example.com", UnsubscribeToken: "tok-99"},
		}, nil)

		m := &mockEmail{}
		agg := Aggregator{email: m, issues: issueRepo, items: itemRepo, subscribers: subs}

		require.NoError(t, agg.SendDigest(t.Context(), date, false))
		require.Len(t, m.reqs, 1)

		issueTag := email.Tag{Name: email.TagIssueID, Value: strconv.FormatInt(stored.ID, 10)}

		t.Log("Subscriber email carries issue and subscriber tags")
		assert.Equal(t, []email.Tag{issueTag, {Name: email.TagSubscriberID, Value: "99"}}, m.reqs[0].Tags)
	})

	t.Run("Prompter Never Called During Send", func(t *testing.T) {
		issueRepo, itemRepo := newTestStores(t)
		date := day("2026-05-01")
		stored, err := issueRepo.Create(t.Context(), news.Issue{
			Slug:    "2026-05-01",
			Subject: "GoDaily - 2026-05-01",
			Status:  news.IssueStatusDraft,
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

		subs := mocksubscriber.NewMockSubscriberRepository(gomock.NewController(t))
		subs.EXPECT().ListActive(gomock.Any()).Return([]domainsubscriber.Subscriber{
			{ID: 1, Email: "reader@example.com", UnsubscribeToken: "tok-1"},
		}, nil)

		m := &mockEmail{}
		p := mockai.NewMockPrompter(gomock.NewController(t))
		// No expectations set on p — any call to the prompter would fail the test.
		agg := Aggregator{email: m, prompter: p, issues: issueRepo, items: itemRepo, subscribers: subs}

		require.NoError(t, agg.SendDigest(t.Context(), date, false))
		assert.True(t, m.called)
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
		_, err = itemRepo.Create(t.Context(), &issue.ID, 1, news.Item{Source: news.SourceMedium, Title: "medium-1", URL: "https://medium.com/1", Published: published})
		require.NoError(t, err)
		_, err = itemRepo.Create(t.Context(), &issue.ID, 2, news.Item{Source: news.SourceGoBlog, Title: "goblog-1", URL: "https://go.dev/1", Published: published})
		require.NoError(t, err)
		_, err = itemRepo.Create(t.Context(), &issue.ID, 3, news.Item{Source: news.SourceGoBlog, Title: "goblog-2", URL: "https://go.dev/2", Published: published})
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
