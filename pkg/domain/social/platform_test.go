// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ainsleyclark/godaily/pkg/domain/social"
)

func TestPlatform_String(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		input social.Platform
		want  string
	}{
		"Bluesky":  {input: social.Bluesky, want: "bluesky"},
		"LinkedIn": {input: social.LinkedIn, want: "linkedin"},
		"Mastodon": {input: social.Mastodon, want: "mastodon"},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, test.input.String())
		})
	}
}
