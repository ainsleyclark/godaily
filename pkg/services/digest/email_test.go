// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"fmt"
	htmltemplate "html/template"
	"strings"
	"testing"
	texttemplate "text/template"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/gateway/email"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// brokenTpl references a field that does not exist on digestData,
// causing template execution to fail at runtime.
const brokenTpl = `{{ .Missing.NotAField }}`

var sendDigestDay = time.Date(2026, time.April, 26, 0, 0, 0, 0, time.UTC)

func sampleSections() []news.SourceItems {
	return []news.SourceItems{
		{
			Source: news.SourceHN,
			Items: []news.Item{
				{
					Source:    news.SourceHN,
					Tag:       news.TagDiscussion,
					Title:     "hello",
					URL:       "https://example.com",
					Score:     42,
					Comments:  7,
					Published: sendDigestDay.Add(time.Hour),
				},
			},
		},
		{
			Source: news.SourceDevTo,
			Items: []news.Item{{
				Source:    news.SourceDevTo,
				Tag:       news.TagArticle,
				Title:     "world",
				URL:       "https://dev.to/world",
				Author:    &news.Author{Name: "gopher"},
				Published: sendDigestDay.Add(time.Hour),
			}},
		},
	}
}

func TestRenderDigest(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		got, err := renderDigest(digestOptions{Day: sendDigestDay, Sources: sampleSections()})
		require.NoError(t, err)
		assert.Contains(t, got.Subject, "April 26, 2026")
		assert.Contains(t, got.HTML, "hello")
		assert.Contains(t, got.Text, "hello")
		assert.NotContains(t, got.HTML, "42 pts")
		assert.Contains(t, got.HTML, "7 comments")
		// Each item should advertise its source via the "Read on" link and
		// the inline mark image (HN has a mark file registered).
		assert.Contains(t, got.HTML, "Read on Hacker News")
		assert.Contains(t, got.HTML, "https://godaily.dev/assets/images/marks/hacker_news.svg")
	})

	t.Run("Intro Subjects Split Into Paragraphs", func(t *testing.T) {
		// A blank line in the intro separates distinct subjects, which must
		// render as two paragraphs in HTML rather than collapsing to one.
		got, err := renderDigest(digestOptions{
			Day:     sendDigestDay,
			Intro:   "First subject lands.\n\nSecond subject ships.",
			Sources: sampleSections(),
		})
		require.NoError(t, err)
		assert.Equal(t, 2, strings.Count(got.HTML, "First subject lands.")+strings.Count(got.HTML, "Second subject ships."))
		// Two distinct <p> blocks carry the intro paragraphs.
		introBlock := `<p style="font-size:14px;color:#3a6880;line-height:1.6;margin:12px 0 0;">`
		assert.Equal(t, 2, strings.Count(got.HTML, introBlock))
		assert.Contains(t, got.Text, "First subject lands.")
		assert.Contains(t, got.Text, "Second subject ships.")
	})

	t.Run("Omits Intro When Empty", func(t *testing.T) {
		got, err := renderDigest(digestOptions{Day: sendDigestDay, Intro: "  \n\n ", Sources: sampleSections()})
		require.NoError(t, err)
		assert.NotContains(t, got.HTML, `line-height:1.6;margin:12px 0 0;`)
	})

	t.Run("Groups By Section", func(t *testing.T) {
		// Two sources, two different sections — HN under Discussions and
		// Go Blog under Articles — must produce two section headings and
		// the section title in the rendered output.
		sources := []news.SourceItems{
			{Source: news.SourceHN, Items: []news.Item{{
				Source: news.SourceHN, Tag: news.TagDiscussion,
				Title: "discuss-me", URL: "https://example.com/d",
				Score: 12, Published: sendDigestDay.Add(time.Hour),
			}}},
			{Source: news.SourceGoBlog, Items: []news.Item{{
				Source: news.SourceGoBlog, Tag: news.TagArticle,
				Title: "article-me", URL: "https://go.dev/blog/a",
				Score: 4, Published: sendDigestDay.Add(time.Hour),
			}}},
		}
		got, err := renderDigest(digestOptions{Day: sendDigestDay, Sources: sources})
		require.NoError(t, err)
		assert.Contains(t, got.HTML, "Discussions")
		assert.Contains(t, got.HTML, "Articles")
		assert.Contains(t, got.HTML, "discuss-me")
		assert.Contains(t, got.HTML, "article-me")
		// Order: SectionTags = [Release, Proposal, Discussion, Article, Video, Trending],
		// so Discussions renders before Articles.
		idxArticles := strings.Index(got.HTML, "Articles")
		idxDiscussions := strings.Index(got.HTML, "Discussions")
		assert.Less(t, idxDiscussions, idxArticles, "Discussions section should render before Articles")
	})

	t.Run("Skips Empty Sections", func(t *testing.T) {
		// Only a release item — no other section headings should appear.
		sources := []news.SourceItems{
			{Source: news.SourceGoRelease, Items: []news.Item{{
				Source: news.SourceGoRelease, Tag: news.TagRelease,
				Title: "Go 1.23 RC1", URL: "https://go.dev/x",
				Score: 4, Published: sendDigestDay.Add(time.Hour),
			}}},
		}
		got, err := renderDigest(digestOptions{Day: sendDigestDay, Sources: sources})
		require.NoError(t, err)
		assert.Contains(t, got.HTML, "Releases")
		assert.NotContains(t, got.HTML, "Discussions")
		assert.NotContains(t, got.HTML, "Articles")
		assert.NotContains(t, got.HTML, "Trending")
	})

	t.Run("Renders All Items In Position Order", func(t *testing.T) {
		// The renderer no longer caps or re-ranks — selection and ordering are
		// build's job (news.SelectForDigest), persisted as issue_id + position.
		// Every linked item renders, in ascending Position order regardless of
		// score, and there is no overflow CTA.
		jobs := []news.Item{
			{
				Source: news.SourceHN, Tag: news.TagJobs,
				Title: "job-low-score-first", URL: "https://example.com/job/a",
				Score: 1, Position: 1, Published: sendDigestDay.Add(time.Hour),
			},
			{
				Source: news.SourceHN, Tag: news.TagJobs,
				Title: "job-high-score-second", URL: "https://example.com/job/b",
				Score: 99, Position: 2, Published: sendDigestDay.Add(time.Hour),
			},
		}
		// Add six more so the section is well over the old cap of 5.
		for i := range 6 {
			jobs = append(jobs, news.Item{
				Source:    news.SourceHN,
				Tag:       news.TagJobs,
				Title:     fmt.Sprintf("job-extra-%d", i),
				URL:       fmt.Sprintf("https://example.com/job/extra/%d", i),
				Score:     float64(i),
				Position:  int64(3 + i),
				Published: sendDigestDay.Add(time.Hour),
			})
		}
		sources := []news.SourceItems{{Source: news.SourceHN, Items: jobs}}

		got, err := renderDigest(digestOptions{Day: sendDigestDay, Sources: sources})
		require.NoError(t, err)

		// No cap: every item renders and no overflow CTA appears.
		assert.Contains(t, got.HTML, "job-extra-5")
		assert.NotContains(t, got.HTML, "more Jobs on GoDaily")
		assert.NotContains(t, got.HTML, "/browse/jobs/")

		// Order follows Position, not Score (the high-score item is second).
		idxLow := strings.Index(got.HTML, "job-low-score-first")
		idxHigh := strings.Index(got.HTML, "job-high-score-second")
		assert.Less(t, idxLow, idxHigh, "items render in Position order, not by score")
	})

	// HTML/Text template subtests mutate package-level htmlTmpl/textTmpl
	// and must run sequentially.
	t.Run("HTML Template Error", func(t *testing.T) {
		orig := htmlTmpl
		htmlTmpl = htmltemplate.Must(htmltemplate.New("digest").Parse(brokenTpl))
		t.Cleanup(func() { htmlTmpl = orig })

		_, err := renderDigest(digestOptions{Day: sendDigestDay, Sources: sampleSections()})
		assert.ErrorContains(t, err, "rendering html")
	})

	t.Run("Text Template Error", func(t *testing.T) {
		orig := textTmpl
		textTmpl = texttemplate.Must(texttemplate.New("digest").Parse(brokenTpl))
		t.Cleanup(func() { textTmpl = orig })

		_, err := renderDigest(digestOptions{Day: sendDigestDay, Sources: sampleSections()})
		assert.ErrorContains(t, err, "rendering text")
	})
}

func TestAggregator_SendDigestHelper(t *testing.T) {
	t.Parallel()

	rendered, err := renderDigest(digestOptions{Day: sendDigestDay, Sources: sampleSections()})
	require.NoError(t, err)

	t.Run("Send Error", func(t *testing.T) {
		t.Parallel()

		m := &mockEmail{err: errors.New("boom")}
		agg := Service{email: m, adminEmailAddress: "to@example.com"}

		err := agg.sendRendered(t.Context(), "to@example.com", rendered, nil)
		assert.True(t, m.called)
		assert.ErrorContains(t, err, "boom")
	})

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		m := &mockEmail{}
		agg := Service{email: m, adminEmailAddress: "to@example.com"}

		err := agg.sendRendered(t.Context(), "to@example.com", rendered, nil)
		require.NoError(t, err)
		require.True(t, m.called)
		assert.Equal(t, "GoDaily <digest@godaily.dev>", m.req.From)
		assert.Equal(t, []string{"to@example.com"}, m.req.To)
		assert.Contains(t, m.req.Subject, "April 26, 2026")
		assert.Contains(t, m.req.Html, "hello")
		assert.Contains(t, m.req.Text, "hello")
	})

	t.Run("Sets List-Unsubscribe Headers For Subscriber", func(t *testing.T) {
		t.Parallel()

		const unsubURL = "https://godaily.dev/api/unsubscribe/?token=abc123"
		subRendered, err := renderDigest(digestOptions{Day: sendDigestDay, Sources: sampleSections(), UnsubscribeURL: unsubURL})
		require.NoError(t, err)

		m := &mockEmail{}
		agg := Service{email: m, adminEmailAddress: "admin@example.com"}

		require.NoError(t, agg.sendRendered(t.Context(), "sub@example.com", subRendered, nil))
		assert.Equal(t, "<"+unsubURL+">", m.req.Headers["List-Unsubscribe"])
		assert.Equal(t, "List-Unsubscribe=One-Click", m.req.Headers["List-Unsubscribe-Post"])
	})

	t.Run("No List-Unsubscribe Headers For Admin", func(t *testing.T) {
		t.Parallel()

		m := &mockEmail{}
		agg := Service{email: m, adminEmailAddress: "admin@example.com"}

		require.NoError(t, agg.sendRendered(t.Context(), "admin@example.com", rendered, nil))
		assert.Empty(t, m.req.Headers)
	})

	t.Run("Attaches Tags To Outbound Email", func(t *testing.T) {
		t.Parallel()

		tags := []email.Tag{
			{Name: email.TagIssueID, Value: "42"},
			{Name: email.TagSubscriberID, Value: "7"},
		}

		m := &mockEmail{}
		agg := Service{email: m, adminEmailAddress: "admin@example.com"}

		require.NoError(t, agg.sendRendered(t.Context(), "sub@example.com", rendered, tags))
		assert.Equal(t, tags, m.req.Tags)
	})
}
