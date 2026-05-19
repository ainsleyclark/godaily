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

package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIsWeekend(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		t    time.Time
		want bool
	}{
		"Saturday": {
			t:    time.Date(2026, 5, 16, 10, 0, 0, 0, time.UTC),
			want: true,
		},
		"Sunday": {
			t:    time.Date(2026, 5, 17, 10, 0, 0, 0, time.UTC),
			want: true,
		},
		"Monday": {
			t:    time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC),
			want: false,
		},
		"Friday": {
			t:    time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC),
			want: false,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, IsWeekend(test.t))
		})
	}
}
