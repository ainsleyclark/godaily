// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

// Platform identifies a social platform.
type Platform string

// Platform name constants. The string values are used as the persisted
// platform field on social_posts rows.
const (
	Bluesky  Platform = "bluesky"
	LinkedIn Platform = "linkedin"
	Mastodon Platform = "mastodon"
)

// String implements fmt.Stringer on Platform.
func (p Platform) String() string {
	return string(p)
}
