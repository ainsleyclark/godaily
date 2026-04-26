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

package source

import (
	"testing"

	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/stretchr/testify/assert"
)

func TestRegisteredSources(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		source news.Source
		want   func(news.Fetcher, error)
	}{
		"DevTo": {
			source: news.SourceDevTo,
			want: func(f news.Fetcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, f)
			},
		},
		"GoBlog": {
			source: news.SourceGoBlog,
			want: func(f news.Fetcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, f)
			},
		},
		"HackerNews": {
			source: news.SourceHN,
			want: func(f news.Fetcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, f)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			got, err := news.Get(test.source)
			test.want(got, err)
		})
	}
}
