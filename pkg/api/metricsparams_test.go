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
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMetricsQuery(t *testing.T) {
	t.Parallel()

	allowedSorts := []string{"click_rate", "open_rate", "sent_at"}

	t.Run("Default limit and sort", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/?", nil)
		got, httpErr := ParseMetricsQuery(r, allowedSorts, "sent_at")
		require.Nil(t, httpErr)
		assert.Equal(t, 10, got.Limit)
		assert.Equal(t, "sent_at", got.Sort)
		assert.Nil(t, got.From)
		assert.Nil(t, got.To)
	})

	t.Run("Explicit limit", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/?limit=25", nil)
		got, httpErr := ParseMetricsQuery(r, allowedSorts, "sent_at")
		require.Nil(t, httpErr)
		assert.Equal(t, 25, got.Limit)
	})

	t.Run("Valid from and to", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/?from=2026-01-01&to=2026-02-01", nil)
		got, httpErr := ParseMetricsQuery(r, allowedSorts, "sent_at")
		require.Nil(t, httpErr)
		require.NotNil(t, got.From)
		require.NotNil(t, got.To)
		assert.Equal(t, "2026-01-01", got.From.Format("2006-01-02"))
		assert.Equal(t, "2026-02-01", got.To.Format("2006-01-02"))
	})

	t.Run("Invalid from date", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/?from=not-a-date", nil)
		_, httpErr := ParseMetricsQuery(r, allowedSorts, "sent_at")
		require.NotNil(t, httpErr)
		assert.Equal(t, 400, httpErr.Status)
	})

	t.Run("Invalid to date", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/?to=2026/01/01", nil)
		_, httpErr := ParseMetricsQuery(r, allowedSorts, "sent_at")
		require.NotNil(t, httpErr)
		assert.Equal(t, 400, httpErr.Status)
	})

	t.Run("From equal to to", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/?from=2026-01-01&to=2026-01-01", nil)
		_, httpErr := ParseMetricsQuery(r, allowedSorts, "sent_at")
		require.NotNil(t, httpErr)
		assert.Equal(t, 400, httpErr.Status)
	})

	t.Run("From after to", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/?from=2026-02-01&to=2026-01-01", nil)
		_, httpErr := ParseMetricsQuery(r, allowedSorts, "sent_at")
		require.NotNil(t, httpErr)
		assert.Equal(t, 400, httpErr.Status)
	})

	t.Run("Period day sets from and to", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/?period=day", nil)
		got, httpErr := ParseMetricsQuery(r, allowedSorts, "sent_at")
		require.Nil(t, httpErr)
		require.NotNil(t, got.From)
		require.NotNil(t, got.To)
		diff := got.To.Sub(*got.From)
		assert.InDelta(t, float64(24*time.Hour), float64(diff), float64(time.Minute))
	})

	t.Run("Period week sets 7-day window", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/?period=week", nil)
		got, httpErr := ParseMetricsQuery(r, allowedSorts, "sent_at")
		require.Nil(t, httpErr)
		require.NotNil(t, got.From)
		diff := got.To.Sub(*got.From)
		assert.InDelta(t, float64(7*24*time.Hour), float64(diff), float64(time.Minute))
	})

	t.Run("Period all leaves bounds nil", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/?period=all", nil)
		got, httpErr := ParseMetricsQuery(r, allowedSorts, "sent_at")
		require.Nil(t, httpErr)
		assert.Nil(t, got.From)
		assert.Nil(t, got.To)
	})

	t.Run("Unknown period", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/?period=quarter", nil)
		_, httpErr := ParseMetricsQuery(r, allowedSorts, "sent_at")
		require.NotNil(t, httpErr)
		assert.Equal(t, 400, httpErr.Status)
	})

	t.Run("Period ignored when from is set", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/?from=2026-01-01&period=week", nil)
		got, httpErr := ParseMetricsQuery(r, allowedSorts, "sent_at")
		require.Nil(t, httpErr)
		// from is set, period is ignored, to stays nil
		require.NotNil(t, got.From)
		assert.Nil(t, got.To)
	})

	t.Run("Valid sort", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/?sort=click_rate", nil)
		got, httpErr := ParseMetricsQuery(r, allowedSorts, "sent_at")
		require.Nil(t, httpErr)
		assert.Equal(t, "click_rate", got.Sort)
	})

	t.Run("Unknown sort", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/?sort=nonsense", nil)
		_, httpErr := ParseMetricsQuery(r, allowedSorts, "sent_at")
		require.NotNil(t, httpErr)
		assert.Equal(t, 400, httpErr.Status)
	})

	t.Run("Limit too low", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/?limit=-1", nil)
		_, httpErr := ParseMetricsQuery(r, allowedSorts, "sent_at")
		require.NotNil(t, httpErr)
		assert.Equal(t, 400, httpErr.Status)
	})

	t.Run("Limit too high", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/?limit=101", nil)
		_, httpErr := ParseMetricsQuery(r, allowedSorts, "sent_at")
		require.NotNil(t, httpErr)
		assert.Equal(t, 400, httpErr.Status)
	})

	t.Run("Limit at max boundary", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/?limit=100", nil)
		got, httpErr := ParseMetricsQuery(r, allowedSorts, "sent_at")
		require.Nil(t, httpErr)
		assert.Equal(t, 100, got.Limit)
	})

	t.Run("ToFilter conversion", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/?limit=20", nil)
		got, httpErr := ParseMetricsQuery(r, allowedSorts, "sent_at")
		require.Nil(t, httpErr)
		f := got.ToFilter()
		assert.Equal(t, 20, f.Limit)
		assert.Nil(t, f.From)
		assert.Nil(t, f.To)
	})
}

func TestHTTPError_Write(t *testing.T) {
	t.Parallel()

	e := &HTTPError{Status: 400, Message: "bad input"}
	w := httptest.NewRecorder()
	e.Write(w)

	assert.Equal(t, 400, w.Code)
}
