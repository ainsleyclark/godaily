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

package prompts

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/pkg/news"
)

type mockPrompter struct {
	raw []byte
	err error
}

func (m *mockPrompter) Prompt(_ context.Context, _, _ string) ([]byte, error) {
	return m.raw, m.err
}

func sampleSections() []news.SourceItems {
	return []news.SourceItems{{
		Source: news.SourceGoBlog,
		Items: []news.Item{{
			Source: news.SourceGoBlog,
			Title:  "Go 1.24 ships",
			URL:    "https://go.dev/blog/go1.24",
			Score:  0.95,
		}},
	}}
}

func validSuggestJSON(post string) []byte {
	raw, _ := json.Marshal(map[string]any{
		"post": post,
		"references": []map[string]string{{
			"title":  "Go 1.24 ships",
			"url":    "https://go.dev/blog/go1.24",
			"source": "go_blog",
		}},
	})
	return raw
}

func TestSuggest(t *testing.T) {
	t.Parallel()

	day := time.Date(2026, time.April, 27, 0, 0, 0, 0, time.UTC)

	tt := map[string]struct {
		p        *mockPrompter
		sections []news.SourceItems
		wantErr  string
		check    func(t *testing.T, s Suggestion)
	}{
		"No Items Returns ErrNoItems": {
			p:        &mockPrompter{raw: validSuggestJSON("x")},
			sections: nil,
			wantErr:  ErrNoItems.Error(),
		},
		"Prompter Error Wrapped": {
			p:        &mockPrompter{err: context.DeadlineExceeded},
			sections: sampleSections(),
			wantErr:  "ai",
		},
		"Parse Error Surfaced": {
			p:        &mockPrompter{raw: []byte("not json")},
			sections: sampleSections(),
			wantErr:  "parse (raw=",
		},
		"OK Populates Date": {
			p:        &mockPrompter{raw: validSuggestJSON("great post")},
			sections: sampleSections(),
			check: func(t *testing.T, s Suggestion) {
				t.Helper()
				assert.Equal(t, "great post", s.Post)
				assert.Equal(t, day, s.Date)
				require.Len(t, s.References, 1)
				assert.Equal(t, news.SourceGoBlog, s.References[0].Source)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := Suggest(context.Background(), test.p, day, test.sections)
			if test.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.wantErr)
				return
			}
			require.NoError(t, err)
			test.check(t, got)
		})
	}
}

func TestParseSuggestionBytes(t *testing.T) {
	t.Parallel()

	validJSON := `{"post":"hello","references":[{"title":"t","url":"u","source":"hacker_news"}]}`

	tt := map[string]struct {
		raw     []byte
		wantErr string
		check   func(t *testing.T, s Suggestion)
	}{
		"Empty Body": {
			raw:     []byte(""),
			wantErr: "empty response body",
		},
		"Invalid JSON": {
			raw:     []byte("not json"),
			wantErr: "parse (raw=",
		},
		"Missing Post": {
			raw:     []byte(`{"post":""}`),
			wantErr: "missing post field",
		},
		"Post Too Long Warns But Returns Post": {
			raw: func() []byte {
				b, _ := json.Marshal(map[string]any{"post": strings.Repeat("a", 281), "references": []any{}})
				return b
			}(),
			check: func(t *testing.T, s Suggestion) {
				t.Helper()
				assert.Equal(t, 281, utf8.RuneCountInString(s.Post), "post must be returned unmodified")
			},
		},
		"Valid": {
			raw: []byte(validJSON),
			check: func(t *testing.T, s Suggestion) {
				t.Helper()
				assert.Equal(t, "hello", s.Post)
				require.Len(t, s.References, 1)
				assert.Equal(t, news.SourceHN, s.References[0].Source)
			},
		},
		"Valid With Fenced JSON": {
			raw: []byte("```json\n" + validJSON + "\n```"),
			check: func(t *testing.T, s Suggestion) {
				t.Helper()
				assert.Equal(t, "hello", s.Post)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := parseSuggestionBytes(test.raw)
			if test.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.wantErr)
				return
			}
			require.NoError(t, err)
			test.check(t, got)
		})
	}
}

func TestStripFences(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		in   string
		want string
	}{
		"No Fence":            {in: `{"a":1}`, want: `{"a":1}`},
		"Whitespace Only":     {in: "  \n  ", want: ""},
		"Json Fence":          {in: "```json\n{\"a\":1}\n```", want: `{"a":1}`},
		"Plain Fence":         {in: "```\n{\"a\":1}\n```", want: `{"a":1}`},
		"Surrounding Spaces":  {in: "  ```json\n{\"a\":1}\n```  ", want: `{"a":1}`},
		"Fence Without Close": {in: "```json\n{\"a\":1}", want: `{"a":1}`},
		"Fence Without Newline (no body)": {
			in: "```json", want: "```json",
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, stripFences(tc.in))
		})
	}
}

func TestSuggestion_Markdown(t *testing.T) {
	t.Parallel()

	s := Suggestion{
		Date: time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC),
		Post: "post text",
		References: []Ref{{
			Title:  "Go 1.24 ships",
			URL:    "https://go.dev/blog/go1.24",
			Source: news.SourceGoBlog,
		}},
	}

	t.Run("With References", func(t *testing.T) {
		t.Parallel()
		md := s.Markdown()
		assert.Contains(t, md, "## Suggested post: 2026-04-27")
		assert.Contains(t, md, "post text")
		assert.Contains(t, md, "### References")
		assert.Contains(t, md, "[Go 1.24 ships](https://go.dev/blog/go1.24) (go_blog)")
	})

	t.Run("Without References", func(t *testing.T) {
		t.Parallel()
		noRef := s
		noRef.References = nil
		md := noRef.Markdown()
		assert.NotContains(t, md, "### References")
	})
}

func TestSuggestion_JSON(t *testing.T) {
	t.Parallel()

	s := Suggestion{
		Date: time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC),
		Post: "post text",
		References: []Ref{{
			Title:  "Go 1.24 ships",
			URL:    "https://go.dev/blog/go1.24",
			Source: news.SourceGoBlog,
		}},
	}

	got, err := s.JSON()
	require.NoError(t, err)

	var round Suggestion
	require.NoError(t, json.Unmarshal(got, &round))
	assert.Equal(t, s, round)
	assert.True(t, strings.Contains(string(got), "\n"), "expected multi-line indented JSON")
}
