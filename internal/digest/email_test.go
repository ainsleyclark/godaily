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
	"testing"
	texttemplate "text/template"
	"time"

	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// brokenTpl references a field that does not exist on digestData,
// causing template execution to fail at runtime.
const brokenTpl = `{{ .Missing.NotAField }}`

var sendDigestDay = time.Date(2026, time.April, 26, 0, 0, 0, 0, time.UTC)

func sampleSections() []news.SourceItems {
	return []news.SourceItems{{
		Source: news.SourceDevTo,
		Items: []news.Item{{
			Title:     "hello",
			URL:       "https://example.com",
			Author:    &news.Author{Name: "gopher"},
			Published: sendDigestDay.Add(time.Hour),
		}},
	}}
}

func TestRenderDigest(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		got, err := renderDigest(sendDigestDay, sampleSections(), nil)
		require.NoError(t, err)
		assert.Contains(t, got.Subject, "April 26, 2026")
		assert.Contains(t, got.HTML, "hello")
		assert.Contains(t, got.Text, "hello")
	})

	// HTML/Text template subtests mutate package-level htmlTmpl/textTmpl
	// and must run sequentially.
	t.Run("HTML Template Error", func(t *testing.T) {
		orig := htmlTmpl
		htmlTmpl = htmltemplate.Must(htmltemplate.New("digest").Parse(brokenTpl))
		t.Cleanup(func() { htmlTmpl = orig })

		_, err := renderDigest(sendDigestDay, sampleSections(), nil)
		assert.ErrorContains(t, err, "rendering html")
	})

	t.Run("Text Template Error", func(t *testing.T) {
		orig := textTmpl
		textTmpl = texttemplate.Must(texttemplate.New("digest").Parse(brokenTpl))
		t.Cleanup(func() { textTmpl = orig })

		_, err := renderDigest(sendDigestDay, sampleSections(), nil)
		assert.ErrorContains(t, err, "rendering text")
	})
}

func TestAggregator_SendDigest(t *testing.T) {
	t.Parallel()

	rendered, err := renderDigest(sendDigestDay, sampleSections(), nil)
	require.NoError(t, err)

	t.Run("Send Error", func(t *testing.T) {
		t.Parallel()

		m := &mockEmail{err: errors.New("boom")}
		agg := Aggregator{email: m, sendToAddress: "to@example.com"}

		err := agg.sendDigest(t.Context(), rendered)
		assert.True(t, m.called)
		assert.ErrorContains(t, err, "boom")
	})

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		m := &mockEmail{}
		agg := Aggregator{email: m, sendToAddress: "to@example.com"}

		err := agg.sendDigest(t.Context(), rendered)
		require.NoError(t, err)
		require.True(t, m.called)
		assert.Equal(t, "noreply@mail.ainsley.dev", m.req.From)
		assert.Equal(t, []string{"to@example.com"}, m.req.To)
		assert.Contains(t, m.req.Subject, "April 26, 2026")
		assert.Contains(t, m.req.Html, "hello")
		assert.Contains(t, m.req.Text, "hello")
	})
}
