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
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/mocks/ai"
)

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

func validSuggestJSON(posts ...string) []byte {
	arr := make([]map[string]any, len(posts))
	for i, post := range posts {
		arr[i] = map[string]any{
			"post": post,
			"references": []map[string]string{{
				"title":  "Go 1.24 ships",
				"url":    "https://go.dev/blog/go1.24",
				"source": "go_blog",
			}},
		}
	}
	raw, _ := json.Marshal(map[string]any{"posts": arr})
	return raw
}

func TestSuggest(t *testing.T) {
	t.Parallel()

	day := time.Date(2026, time.April, 27, 0, 0, 0, 0, time.UTC)

	tt := map[string]struct {
		raw       []byte
		promptErr error
		sections  []news.SourceItems
		wantErr   string
		check     func(t *testing.T, s Suggestion)
	}{
		"No Items Returns ErrNoItems": {
			raw:      validSuggestJSON("x"),
			sections: nil,
			wantErr:  ErrNoItems.Error(),
		},
		"Prompter Error Wrapped": {
			promptErr: context.DeadlineExceeded,
			sections:  sampleSections(),
			wantErr:   "ai",
		},
		"Parse Error Surfaced": {
			raw:      []byte("not json"),
			sections: sampleSections(),
			wantErr:  "parse (raw=",
		},
		"OK Populates Date": {
			raw:      validSuggestJSON("first post", "second post", "third post"),
			sections: sampleSections(),
			check: func(t *testing.T, s Suggestion) {
				t.Helper()
				require.Len(t, s.Posts, 3)
				assert.Equal(t, "first post", s.Posts[0].Text)
				assert.Equal(t, day, s.Date)
				require.Len(t, s.Posts[0].References, 1)
				assert.Equal(t, news.SourceGoBlog, s.Posts[0].References[0].Source)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			p := mockai.NewMockPrompter(gomock.NewController(t))
			if len(test.sections) > 0 {
				p.EXPECT().Prompt(gomock.Any(), gomock.Any(), gomock.Any()).Return(test.raw, test.promptErr)
			}
			got, err := Suggest(context.Background(), p, day, test.sections)
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

	validJSON := `{"posts":[{"post":"hello","references":[{"title":"t","url":"u","source":"hacker_news"}]}]}`

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
		"No Posts": {
			raw:     []byte(`{"posts":[]}`),
			wantErr: "missing posts field",
		},
		"Empty Post Text": {
			raw:     []byte(`{"posts":[{"post":"ok"},{"post":""}]}`),
			wantErr: "post 2: missing post field",
		},
		"Post Too Long Warns But Returns Post": {
			raw: func() []byte {
				b, _ := json.Marshal(map[string]any{
					"posts": []map[string]any{{"post": strings.Repeat("a", 281), "references": []any{}}},
				})
				return b
			}(),
			check: func(t *testing.T, s Suggestion) {
				t.Helper()
				require.Len(t, s.Posts, 1)
				assert.Equal(t, 281, utf8.RuneCountInString(s.Posts[0].Text), "post must be returned unmodified")
			},
		},
		"Valid": {
			raw: []byte(validJSON),
			check: func(t *testing.T, s Suggestion) {
				t.Helper()
				require.Len(t, s.Posts, 1)
				assert.Equal(t, "hello", s.Posts[0].Text)
				require.Len(t, s.Posts[0].References, 1)
				assert.Equal(t, news.SourceHN, s.Posts[0].References[0].Source)
			},
		},
		"Valid With Fenced JSON": {
			raw: []byte("```json\n" + validJSON + "\n```"),
			check: func(t *testing.T, s Suggestion) {
				t.Helper()
				require.Len(t, s.Posts, 1)
				assert.Equal(t, "hello", s.Posts[0].Text)
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

func TestSuggestion_Markdown(t *testing.T) {
	t.Parallel()

	s := Suggestion{
		Date: time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC),
		Posts: []Post{{
			Text: "post text",
			References: []Ref{{
				Title:  "Go 1.24 ships",
				URL:    "https://go.dev/blog/go1.24",
				Source: news.SourceGoBlog,
			}},
		}},
	}

	t.Run("With References", func(t *testing.T) {
		t.Parallel()

		md := s.Markdown()
		assert.Contains(t, md, "## Suggested posts: 2026-04-27")
		assert.Contains(t, md, "### Post 1")
		assert.Contains(t, md, "post text")
		assert.Contains(t, md, "#### References")
		assert.Contains(t, md, "[Go 1.24 ships](https://go.dev/blog/go1.24) (go_blog)")
	})

	t.Run("Without References", func(t *testing.T) {
		t.Parallel()

		noRef := Suggestion{Date: s.Date, Posts: []Post{{Text: "post text"}}}
		md := noRef.Markdown()
		assert.NotContains(t, md, "#### References")
	})
}

func TestSuggestion_JSON(t *testing.T) {
	t.Parallel()

	s := Suggestion{
		Date: time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC),
		Posts: []Post{{
			Text: "post text",
			References: []Ref{{
				Title:  "Go 1.24 ships",
				URL:    "https://go.dev/blog/go1.24",
				Source: news.SourceGoBlog,
			}},
		}},
	}

	got, err := s.JSON()
	require.NoError(t, err)

	var round Suggestion
	require.NoError(t, json.Unmarshal(got, &round))
	assert.Equal(t, s, round)
	assert.True(t, strings.Contains(string(got), "\n"), "expected multi-line indented JSON")
}
