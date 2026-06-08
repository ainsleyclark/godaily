// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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
		// the previous complete ISO week (Mon 00:00 → next Mon 00:00,
		// UTC) — the Monday recap window covering the week that just
		// finished.
		Window time.Duration
		// MinItems is the floor below which Top returns its zero value
		// (so the caller can no-op cleanly). Defaults to 0 — every
		// result kept.
		MinItems int
	}
)

// HasItems reports whether the recap has any ranked items at all.
func (t Top) HasItems() bool { return len(t.Items) > 0 }
