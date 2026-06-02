// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBuildWindow(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		day       time.Time
		wantStart time.Time
		wantEnd   time.Time
	}{
		"Monday reaches back across the weekend": {
			day:       time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), // Monday
			wantStart: time.Date(2026, 5, 29, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		},
		"Weekday covers the previous day": {
			day:       time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC), // Tuesday
			wantStart: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC),
		},
		"Truncates a non-midnight input": {
			day:       time.Date(2026, 6, 2, 13, 45, 0, 0, time.UTC),
			wantStart: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC),
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			start, end := BuildWindow(test.day)
			assert.Equal(t, test.wantStart, start)
			assert.Equal(t, test.wantEnd, end)
		})
	}
}
