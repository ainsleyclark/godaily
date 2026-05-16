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

package ai

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/pkg/news"
)

func sampleSuggestion() Suggestion {
	return Suggestion{
		Date: time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC),
		Post: "post text",
		References: []Ref{{
			Title:  "Go 1.24 ships",
			URL:    "https://go.dev/blog/go1.24",
			Source: news.SourceGoBlog,
		}},
	}
}

func TestSuggestion_Markdown(t *testing.T) {
	t.Parallel()

	t.Run("With References", func(t *testing.T) {
		t.Parallel()
		md := sampleSuggestion().Markdown()
		assert.Contains(t, md, "## Suggested post: 2026-04-27")
		assert.Contains(t, md, "post text")
		assert.Contains(t, md, "### References")
		assert.Contains(t, md, "[Go 1.24 ships](https://go.dev/blog/go1.24) (go_blog)")
	})

	t.Run("Without References", func(t *testing.T) {
		t.Parallel()
		s := sampleSuggestion()
		s.References = nil
		md := s.Markdown()
		assert.NotContains(t, md, "### References")
	})
}

func TestSuggestion_JSON(t *testing.T) {
	t.Parallel()

	got, err := sampleSuggestion().JSON()
	require.NoError(t, err)

	var round Suggestion
	require.NoError(t, json.Unmarshal(got, &round))
	assert.Equal(t, sampleSuggestion(), round)
	assert.True(t, strings.Contains(string(got), "\n"), "expected multi-line indented JSON")
}

// makeTextMessage builds a minimal *anthropic.Message with one or more
// text blocks. Used by parseResponse tests.
func makeTextMessage(parts ...string) *anthropic.Message {
	msg := &anthropic.Message{}
	for _, p := range parts {
		msg.Content = append(msg.Content, anthropic.ContentBlockUnion{
			Type: "text",
			Text: p,
		})
	}
	return msg
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

func TestParseResponse(t *testing.T) {
	t.Parallel()

	validJSON := `{"post":"hello","references":[{"title":"t","url":"u","source":"hacker_news"}]}`

	tt := map[string]struct {
		msg     *anthropic.Message
		wantErr string
		check   func(t *testing.T, s Suggestion)
	}{
		"Nil Message": {
			msg:     nil,
			wantErr: "nil message",
		},
		"Empty Body": {
			msg:     &anthropic.Message{},
			wantErr: "empty response body",
		},
		"Non-Text Block Ignored": {
			msg: &anthropic.Message{Content: []anthropic.ContentBlockUnion{
				{Type: "tool_use", Text: ""},
			}},
			wantErr: "empty response body",
		},
		"Invalid JSON": {
			msg:     makeTextMessage("not json"),
			wantErr: "parse (raw=",
		},
		"Missing Post": {
			msg:     makeTextMessage(`{"post":""}`),
			wantErr: "missing post field",
		},
		"Post Too Long Warns But Returns Post": {
			msg: makeTextMessage(`{"post":"` + strings.Repeat("a", 281) + `"}`),
			check: func(t *testing.T, s Suggestion) {
				t.Helper()
				assert.Equal(t, 281, utf8.RuneCountInString(s.Post), "post must be returned unmodified")
			},
		},
		"Valid": {
			msg: makeTextMessage(validJSON),
			check: func(t *testing.T, s Suggestion) {
				t.Helper()
				assert.Equal(t, "hello", s.Post)
				require.Len(t, s.References, 1)
				assert.Equal(t, news.SourceHN, s.References[0].Source)
			},
		},
		"Valid With Fenced JSON": {
			msg: makeTextMessage("```json\n" + validJSON + "\n```"),
			check: func(t *testing.T, s Suggestion) {
				t.Helper()
				assert.Equal(t, "hello", s.Post)
			},
		},
		"Multiple Text Blocks Concatenated": {
			msg: makeTextMessage(`{"post":`, `"y","references":[]}`),
			check: func(t *testing.T, s Suggestion) {
				t.Helper()
				assert.Equal(t, "y", s.Post)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := parseResponse(test.msg)
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
