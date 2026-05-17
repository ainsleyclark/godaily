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
)

func validDigestJSON(title, intro string) []byte {
	raw, _ := json.Marshal(DigestMeta{Title: title, Intro: intro})
	return raw
}

func TestSynthesise(t *testing.T) {
	t.Parallel()

	day := time.Date(2026, time.April, 27, 0, 0, 0, 0, time.UTC)

	tt := map[string]struct {
		p        *mockPrompter
		sections interface{}
		wantErr  string
		check    func(t *testing.T, m DigestMeta)
	}{
		"No Items Returns ErrNoItems": {
			p:        &mockPrompter{raw: validDigestJSON("t", "i")},
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
		"OK Returns Title And Intro": {
			p:        &mockPrompter{raw: validDigestJSON("Go 1.24 lands", "Goroutines got faster.")},
			sections: sampleSections(),
			check: func(t *testing.T, m DigestMeta) {
				t.Helper()
				assert.Equal(t, "Go 1.24 lands", m.Title)
				assert.Equal(t, "Goroutines got faster.", m.Intro)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var sections []interface{}
			_ = sections
			got, err := Synthesise(context.Background(), test.p, day, sampleSections())
			if test.sections == nil {
				got, err = Synthesise(context.Background(), test.p, day, nil)
			}
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

func TestParseDigestBytes(t *testing.T) {
	t.Parallel()

	validJSON := `{"title":"Go 1.24 lands","intro":"Goroutines got faster."}`

	tt := map[string]struct {
		raw     []byte
		wantErr string
		check   func(t *testing.T, m DigestMeta)
	}{
		"Empty Body": {
			raw:     []byte(""),
			wantErr: "empty response body",
		},
		"Invalid JSON": {
			raw:     []byte("not json"),
			wantErr: "parse (raw=",
		},
		"Missing Title": {
			raw:     []byte(`{"title":"","intro":"something"}`),
			wantErr: "missing title field",
		},
		"Missing Intro": {
			raw:     []byte(`{"title":"Go 1.24 lands","intro":""}`),
			wantErr: "missing intro field",
		},
		"Title Too Long Warns But Returns": {
			raw: func() []byte {
				b, _ := json.Marshal(DigestMeta{Title: strings.Repeat("a", 81), Intro: "x"})
				return b
			}(),
			check: func(t *testing.T, m DigestMeta) {
				t.Helper()
				assert.Equal(t, 81, utf8.RuneCountInString(m.Title))
			},
		},
		"Valid": {
			raw: []byte(validJSON),
			check: func(t *testing.T, m DigestMeta) {
				t.Helper()
				assert.Equal(t, "Go 1.24 lands", m.Title)
				assert.Equal(t, "Goroutines got faster.", m.Intro)
			},
		},
		"Valid With Fenced JSON": {
			raw: []byte("```json\n" + validJSON + "\n```"),
			check: func(t *testing.T, m DigestMeta) {
				t.Helper()
				assert.Equal(t, "Go 1.24 lands", m.Title)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := parseDigestBytes(test.raw)
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
