// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"strings"

	"github.com/ainsleyclark/godaily/pkg/domain/social"
)

// kindLabel renders a PostKind as a human title for card headings, e.g.
// "new_source" -> "New source".
func kindLabel(k social.PostKind) string {
	s := strings.ReplaceAll(string(k), "_", " ")
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// plural returns one when n == 1 and many otherwise.
func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
