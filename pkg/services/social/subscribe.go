// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"unicode/utf8"

	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/utm"
)

// appendSubscribeLine appends a UTM-tagged GoDaily subscribe URL to text.
// Bluesky is skipped (300-char limit leaves no headroom). If appending
// would exceed the platform char limit, the original text is returned unchanged.
func appendSubscribeLine(text string, plat social.Platform, campaign string) string {
	charLimits := map[social.Platform]int{
		social.LinkedIn: 1300,
		social.Mastodon: 500,
	}
	limit, ok := charLimits[plat]
	if !ok {
		return text
	}
	subscribeURL := utm.Tag(env.AppURL+"/", "social-"+string(plat), "social", campaign)
	full := text + "\n\nSubscribe: " + subscribeURL
	if utf8.RuneCountInString(full) > limit {
		return text
	}
	return full
}
