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
	"github.com/stretchr/testify/require"
)

func TestParseDateWindow(t *testing.T) {
	t.Parallel()

	t.Run("Empty inputs leave bounds nil", func(t *testing.T) {
		t.Parallel()
		from, to, err := ParseDateWindow("", "", "")
		require.NoError(t, err)
		assert.Nil(t, from)
		assert.Nil(t, to)
	})

	t.Run("Valid from and to", func(t *testing.T) {
		t.Parallel()
		from, to, err := ParseDateWindow("2026-01-01", "2026-02-01", "")
		require.NoError(t, err)
		require.NotNil(t, from)
		require.NotNil(t, to)
		assert.Equal(t, "2026-01-01", from.Format("2006-01-02"))
		assert.Equal(t, "2026-02-01", to.Format("2006-01-02"))
	})

	t.Run("Invalid from date", func(t *testing.T) {
		t.Parallel()
		_, _, err := ParseDateWindow("not-a-date", "", "")
		require.Error(t, err)
	})

	t.Run("Invalid to date", func(t *testing.T) {
		t.Parallel()
		_, _, err := ParseDateWindow("", "2026/01/01", "")
		require.Error(t, err)
	})

	t.Run("From equal to to", func(t *testing.T) {
		t.Parallel()
		_, _, err := ParseDateWindow("2026-01-01", "2026-01-01", "")
		require.Error(t, err)
	})

	t.Run("From after to", func(t *testing.T) {
		t.Parallel()
		_, _, err := ParseDateWindow("2026-02-01", "2026-01-01", "")
		require.Error(t, err)
	})

	t.Run("Period day sets from and to", func(t *testing.T) {
		t.Parallel()
		from, to, err := ParseDateWindow("", "", "day")
		require.NoError(t, err)
		require.NotNil(t, from)
		require.NotNil(t, to)
		diff := to.Sub(*from)
		assert.InDelta(t, float64(24*time.Hour), float64(diff), float64(time.Minute))
	})

	t.Run("Period week sets 7-day window", func(t *testing.T) {
		t.Parallel()
		from, to, err := ParseDateWindow("", "", "week")
		require.NoError(t, err)
		require.NotNil(t, from)
		diff := to.Sub(*from)
		assert.InDelta(t, float64(7*24*time.Hour), float64(diff), float64(time.Minute))
	})

	t.Run("Period all leaves bounds nil", func(t *testing.T) {
		t.Parallel()
		from, to, err := ParseDateWindow("", "", "all")
		require.NoError(t, err)
		assert.Nil(t, from)
		assert.Nil(t, to)
	})

	t.Run("Unknown period", func(t *testing.T) {
		t.Parallel()
		_, _, err := ParseDateWindow("", "", "quarter")
		require.Error(t, err)
	})

	t.Run("Period ignored when from is set", func(t *testing.T) {
		t.Parallel()
		from, to, err := ParseDateWindow("2026-01-01", "", "week")
		require.NoError(t, err)
		require.NotNil(t, from)
		assert.Nil(t, to)
	})

	t.Run("Unknown period still rejected when from is set", func(t *testing.T) {
		t.Parallel()
		_, _, err := ParseDateWindow("2026-01-01", "", "quarter")
		require.Error(t, err)
	})
}
