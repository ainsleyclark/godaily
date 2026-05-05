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
	"github.com/ainsleyclark/godaily/internal/synth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
