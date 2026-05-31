// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package utm tags outbound, GoDaily-owned links with UTM query
// parameters so Plausible can attribute new subscribers to the channel
// that brought them in. See docs/features/share-attribution.md.
package utm

import "net/url"

// Tag appends utm_source / utm_medium / utm_campaign to rawURL and
// returns the re-encoded URL. Existing query parameters are preserved
// and any utm_* value already present is overwritten, so tagging is
// idempotent. Empty source/medium/campaign fields are skipped.
//
// If rawURL cannot be parsed it is returned unchanged — a tagged link is
// a nice-to-have for analytics, never worth breaking the link over.
func Tag(rawURL, source, medium, campaign string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	q := u.Query()
	for key, val := range map[string]string{
		"utm_source":   source,
		"utm_medium":   medium,
		"utm_campaign": campaign,
	} {
		if val == "" {
			continue
		}
		q.Set(key, val)
	}
	u.RawQuery = q.Encode()

	return u.String()
}
