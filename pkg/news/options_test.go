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

package news

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListOptions_Limit(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		input ListOptions
		want  int64
	}{
		"Zero value returns all":     {input: ListOptions{}, want: 10000},
		"Page zero returns all":      {input: ListOptions{Page: 0, PerPage: 50}, want: 10000},
		"Explicit per page":          {input: ListOptions{Page: 1, PerPage: 50}, want: 50},
		"Zero per page uses default": {input: ListOptions{Page: 1, PerPage: 0}, want: defaultPerPage},
		"Negative per page defaults": {input: ListOptions{Page: 1, PerPage: -1}, want: defaultPerPage},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := test.input.Limit()
			assert.Equal(t, test.want, got)
		})
	}
}

func TestListOptions_Offset(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		input ListOptions
		want  int64
	}{
		"Zero value":    {input: ListOptions{}, want: 0},
		"Page 1":        {input: ListOptions{Page: 1, PerPage: 20}, want: 0},
		"Page 2":        {input: ListOptions{Page: 2, PerPage: 20}, want: 20},
		"Page 3":        {input: ListOptions{Page: 3, PerPage: 10}, want: 20},
		"Page 5":        {input: ListOptions{Page: 5, PerPage: 25}, want: 100},
		"Zero per page": {input: ListOptions{Page: 2, PerPage: 0}, want: defaultPerPage},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := test.input.Offset()
			assert.Equal(t, test.want, got)
		})
	}
}
