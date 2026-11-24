// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ainsleyclark/godaily/pkg/env"
)

func TestNewLinkedIn(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		input env.Config
		want  bool
	}{
		"Nothing set":           {input: env.Config{}, want: false},
		"Token without org URN": {input: env.Config{LinkedInOAuthToken: "tok"}, want: false},
		"Static token":          {input: env.Config{LinkedInOAuthToken: "tok", LinkedInOrgURN: "urn"}, want: true},
		"Refresh only": {
			input: env.Config{
				LinkedInOrgURN:       "urn",
				LinkedInRefreshToken: "rt",
				LinkedInClientID:     "id",
				LinkedInClientSecret: "secret",
			},
			want: true,
		},
		"Partial refresh": {
			input: env.Config{LinkedInOrgURN: "urn", LinkedInRefreshToken: "rt", LinkedInClientID: "id"},
			want:  false,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			li, ok := NewLinkedIn(test.input, "")
			assert.Equal(t, test.want, ok)
			assert.Equal(t, test.want, li != nil)
		})
	}
}
