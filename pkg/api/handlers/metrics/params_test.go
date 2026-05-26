// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDateWindow(t *testing.T) {
	t.Parallel()

	t.Run("Empty inputs leave bounds nil", func(t *testing.T) {
		t.Parallel()
		from, to, err := parseDateWindow("", "", "")
		require.NoError(t, err)
		assert.Nil(t, from)
		assert.Nil(t, to)
	})

	t.Run("Valid from and to", func(t *testing.T) {
		t.Parallel()
		from, to, err := parseDateWindow("2026-01-01", "2026-02-01", "")
		require.NoError(t, err)
		require.NotNil(t, from)
		require.NotNil(t, to)
		assert.Equal(t, "2026-01-01", from.Format("2006-01-02"))
		assert.Equal(t, "2026-02-01", to.Format("2006-01-02"))
	})

	t.Run("Invalid from date", func(t *testing.T) {
		t.Parallel()
		_, _, err := parseDateWindow("not-a-date", "", "")
		require.Error(t, err)
	})

	t.Run("Invalid to date", func(t *testing.T) {
		t.Parallel()
		_, _, err := parseDateWindow("", "2026/01/01", "")
		require.Error(t, err)
	})

	t.Run("From equal to to", func(t *testing.T) {
		t.Parallel()
		_, _, err := parseDateWindow("2026-01-01", "2026-01-01", "")
		require.Error(t, err)
	})

	t.Run("From after to", func(t *testing.T) {
		t.Parallel()
		_, _, err := parseDateWindow("2026-02-01", "2026-01-01", "")
		require.Error(t, err)
	})

	t.Run("Period day sets from and to", func(t *testing.T) {
		t.Parallel()
		from, to, err := parseDateWindow("", "", "day")
		require.NoError(t, err)
		require.NotNil(t, from)
		require.NotNil(t, to)
		diff := to.Sub(*from)
		assert.InDelta(t, float64(24*time.Hour), float64(diff), float64(time.Minute))
	})

	t.Run("Period week sets 7-day window", func(t *testing.T) {
		t.Parallel()
		from, to, err := parseDateWindow("", "", "week")
		require.NoError(t, err)
		require.NotNil(t, from)
		diff := to.Sub(*from)
		assert.InDelta(t, float64(7*24*time.Hour), float64(diff), float64(time.Minute))
	})

	t.Run("Period all leaves bounds nil", func(t *testing.T) {
		t.Parallel()
		from, to, err := parseDateWindow("", "", "all")
		require.NoError(t, err)
		assert.Nil(t, from)
		assert.Nil(t, to)
	})

	t.Run("Unknown period", func(t *testing.T) {
		t.Parallel()
		_, _, err := parseDateWindow("", "", "quarter")
		require.Error(t, err)
	})

	t.Run("Period ignored when from is set", func(t *testing.T) {
		t.Parallel()
		from, to, err := parseDateWindow("2026-01-01", "", "week")
		require.NoError(t, err)
		require.NotNil(t, from)
		assert.Nil(t, to)
	})

	t.Run("Unknown period still rejected when from is set", func(t *testing.T) {
		t.Parallel()
		_, _, err := parseDateWindow("2026-01-01", "", "quarter")
		require.Error(t, err)
	})
}
