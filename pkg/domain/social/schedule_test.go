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

package social_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ainsleyclark/godaily/pkg/domain/social"
)

func TestPickSlot(t *testing.T) {
	t.Parallel()

	t.Run("Stable across calls for same date", func(t *testing.T) {
		t.Parallel()

		d := time.Date(2026, time.May, 20, 0, 0, 0, 0, time.UTC)
		first := social.PickSlot(d)
		for i := 0; i < 10; i++ {
			assert.Equal(t, first, social.PickSlot(d))
		}
	})

	t.Run("Always within 0..SlotsPerHour-1", func(t *testing.T) {
		t.Parallel()

		start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
		for i := 0; i < 365; i++ {
			d := start.AddDate(0, 0, i)
			slot := social.PickSlot(d)
			assert.GreaterOrEqual(t, slot, 0)
			assert.Less(t, slot, social.CronSlotsPerHour)
		}
	})

	t.Run("Different dates can produce different slots", func(t *testing.T) {
		t.Parallel()

		start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
		seen := make(map[int]bool)
		for i := 0; i < 60; i++ {
			seen[social.PickSlot(start.AddDate(0, 0, i))] = true
		}
		assert.GreaterOrEqual(t, len(seen), 4, "expected slot distribution to vary across days")
	})

	t.Run("Date-only — time of day is irrelevant", func(t *testing.T) {
		t.Parallel()

		d := time.Date(2026, time.May, 20, 0, 0, 0, 0, time.UTC)
		dPM := time.Date(2026, time.May, 20, 23, 59, 59, 0, time.UTC)
		assert.Equal(t, social.PickSlot(d), social.PickSlot(dPM))
	})
}

func TestShouldRun(t *testing.T) {
	t.Parallel()

	d := time.Date(2026, time.May, 20, 0, 0, 0, 0, time.UTC)
	slot := social.PickSlot(d)

	t.Run("True for the picked minute slot", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2026, time.May, 20, 11, slot*10+3, 0, 0, time.UTC)
		assert.True(t, social.ShouldRun(now, d))
	})

	t.Run("False for other slots", func(t *testing.T) {
		t.Parallel()

		other := (slot + 1) % social.CronSlotsPerHour
		now := time.Date(2026, time.May, 20, 11, other*10+3, 0, 0, time.UTC)
		assert.False(t, social.ShouldRun(now, d))
	})

	t.Run("All 6 minute slots map to exactly one match in an hour", func(t *testing.T) {
		t.Parallel()

		matches := 0
		for s := 0; s < social.CronSlotsPerHour; s++ {
			now := time.Date(2026, time.May, 20, 11, s*10, 0, 0, time.UTC)
			if social.ShouldRun(now, d) {
				matches++
			}
		}
		assert.Equal(t, 1, matches)
	})
}
