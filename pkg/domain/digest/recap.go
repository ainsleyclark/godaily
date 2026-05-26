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

package digest

import (
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
)

type (
	// Period describes the [Start, End) window covered by a recap.
	Period struct {
		Start time.Time
		End   time.Time
		// Label is an ISO-week style identifier used for idempotency
		// keys, e.g. "2026-W21". Stable across reruns within the same
		// week.
		Label string
	}
	// RankedItem is one entry in a recap, paired with its click count.
	RankedItem struct {
		engagement.ItemMetrics
	}
	// Top is the recap dataset.
	Top struct {
		Period Period
		Items  []RankedItem
	}
	// TopOptions tunes a Top call.
	TopOptions struct {
		// N caps the returned items. Zero means the service default (3).
		N int
		// Window is the lookback duration ending at "now". Zero means
		// "since Monday 00:00 UTC of now's week" — the natural Mon→Fri
		// recap window.
		Window time.Duration
		// MinItems is the floor below which Top returns its zero value
		// (so the caller can no-op cleanly). Defaults to 0 — every
		// result kept.
		MinItems int
	}
)

// HasItems reports whether the recap has any ranked items at all.
func (t Top) HasItems() bool { return len(t.Items) > 0 }
