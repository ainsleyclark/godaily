// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package utm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTag(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		rawURL   string
		source   string
		medium   string
		campaign string
		want     string
	}{
		"No existing query": {
			rawURL:   "https://godaily.dev/issues/2026-05-31/",
			source:   "email",
			medium:   "email",
			campaign: "daily-digest",
			want:     "https://godaily.dev/issues/2026-05-31/?utm_campaign=daily-digest&utm_medium=email&utm_source=email",
		},
		"Existing query is preserved": {
			rawURL:   "https://godaily.dev/?ref=foo",
			source:   "social-bluesky",
			medium:   "social",
			campaign: "cta",
			want:     "https://godaily.dev/?ref=foo&utm_campaign=cta&utm_medium=social&utm_source=social-bluesky",
		},
		"Re-tagging overwrites and is idempotent": {
			rawURL:   "https://godaily.dev/?utm_source=old&utm_medium=old&utm_campaign=old",
			source:   "linkedin",
			medium:   "share",
			campaign: "issue-share",
			want:     "https://godaily.dev/?utm_campaign=issue-share&utm_medium=share&utm_source=linkedin",
		},
		"Empty fields are skipped": {
			rawURL:   "https://godaily.dev/",
			source:   "copy",
			medium:   "",
			campaign: "",
			want:     "https://godaily.dev/?utm_source=copy",
		},
		"Unparseable url returned unchanged": {
			rawURL:   "://not a url",
			source:   "email",
			medium:   "email",
			campaign: "daily-digest",
			want:     "://not a url",
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := Tag(test.rawURL, test.source, test.medium, test.campaign)
			assert.Equal(t, test.want, got)
		})
	}
}
