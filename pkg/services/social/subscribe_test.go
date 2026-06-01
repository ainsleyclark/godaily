// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"

	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/env"
)

func TestAppendSubscribeLine(t *testing.T) {
	t.Parallel()

	const base = "Some post text."

	t.Run("Bluesky is unchanged", func(t *testing.T) {
		t.Parallel()

		got := appendSubscribeLine(base, social.Bluesky, "new_source")
		assert.Equal(t, base, got)
	})

	t.Run("LinkedIn appends subscribe line with correct UTM params", func(t *testing.T) {
		t.Parallel()

		got := appendSubscribeLine(base, social.LinkedIn, "new_source")
		assert.True(t, strings.HasPrefix(got, base+"\n\nSubscribe: "))
		assert.Contains(t, got, env.AppURL)
		assert.Contains(t, got, "utm_source=social-linkedin")
		assert.Contains(t, got, "utm_medium=social")
		assert.Contains(t, got, "utm_campaign=new_source")
	})

	t.Run("Mastodon appends subscribe line when it fits", func(t *testing.T) {
		t.Parallel()

		got := appendSubscribeLine(base, social.Mastodon, "recap")
		assert.True(t, strings.HasPrefix(got, base+"\n\nSubscribe: "))
		assert.Contains(t, got, "utm_source=social-mastodon")
		assert.Contains(t, got, "utm_campaign=recap")
		assert.LessOrEqual(t, utf8.RuneCountInString(got), 500)
	})

	t.Run("Mastodon is unchanged when appending would exceed limit", func(t *testing.T) {
		t.Parallel()

		long := strings.Repeat("x", 480)
		got := appendSubscribeLine(long, social.Mastodon, "recap")
		assert.Equal(t, long, got, "must return original when appended text exceeds 500 chars")
	})

	t.Run("LinkedIn campaign is recap for recap kind", func(t *testing.T) {
		t.Parallel()

		got := appendSubscribeLine(base, social.LinkedIn, "recap")
		assert.Contains(t, got, "utm_campaign=recap")
	})
}
