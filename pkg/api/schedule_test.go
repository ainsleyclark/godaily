// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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

func TestIsRotationDay(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		t    time.Time
		want bool
	}{
		"Monday":    {t: time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC), want: true},
		"Tuesday":   {t: time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC), want: false},
		"Wednesday": {t: time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC), want: true},
		"Thursday":  {t: time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC), want: false},
		"Friday":    {t: time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC), want: true},
		"Saturday":  {t: time.Date(2026, 5, 23, 10, 0, 0, 0, time.UTC), want: false},
		"Sunday":    {t: time.Date(2026, 5, 24, 10, 0, 0, 0, time.UTC), want: false},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, IsRotationDay(test.t))
		})
	}
}
