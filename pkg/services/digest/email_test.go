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
	htmltemplate "html/template"
	"strings"
	"testing"
	texttemplate "text/template"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
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
		agg := Aggregator{email: m, adminEmailAddress: "to@example.com"}

		err := agg.sendRendered(t.Context(), "to@example.com", rendered)
		assert.True(t, m.called)
		assert.ErrorContains(t, err, "boom")
	})

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		m := &mockEmail{}
		agg := Aggregator{email: m, adminEmailAddress: "to@example.com"}

		err := agg.sendRendered(t.Context(), "to@example.com", rendered)
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
		agg := Aggregator{email: m, adminEmailAddress: "admin@example.com"}

		require.NoError(t, agg.sendRendered(t.Context(), "sub@example.com", subRendered))
		assert.Equal(t, "<"+unsubURL+">", m.req.Headers["List-Unsubscribe"])
		assert.Equal(t, "List-Unsubscribe=One-Click", m.req.Headers["List-Unsubscribe-Post"])
	})

	t.Run("No List-Unsubscribe Headers For Admin", func(t *testing.T) {
		t.Parallel()

		m := &mockEmail{}
		agg := Aggregator{email: m, adminEmailAddress: "admin@example.com"}

		require.NoError(t, agg.sendRendered(t.Context(), "admin@example.com", rendered))
		assert.Empty(t, m.req.Headers)
	})
}
