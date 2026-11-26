// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build integration

package candidates_test

import (
	"context"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/data"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/gateway/meetup"
	"github.com/ainsleyclark/godaily/pkg/mocks/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/candidates"
)

// TestCommunity_Integration runs the real candidate (embedded YAML + live
// Meetup feeds) for today and the next four Wednesdays with nothing posted
// yet, and checks every rendered post is complete and fits Bluesky's
// limit. Weeks with nothing timely are logged, not failed — quiet weeks
// are expected.
func TestCommunity_Integration(t *testing.T) {
	posts := mocksocial.NewMockPostRepository(gomock.NewController(t))
	allowAll(posts)
	c := candidates.NewCommunity(data.Conferences, data.Meetups, posts, meetup.New())

	// Today (Eligible doesn't check the weekday, so this is the run most
	// likely to have a live event to render) plus the next four Wednesdays.
	today := time.Now().UTC().Truncate(24 * time.Hour).Add(10 * time.Hour)
	wed := today.AddDate(0, 0, 1)
	for wed.Weekday() != time.Wednesday {
		wed = wed.AddDate(0, 0, 1)
	}
	dates := []time.Time{today}
	for i := 0; i < 4; i++ {
		dates = append(dates, wed.AddDate(0, 0, 7*i))
	}

	for _, now := range dates {
		t.Run(now.Format(time.DateOnly), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			cctx, ok, err := c.Eligible(ctx, now)
			require.NoError(t, err)
			if !ok {
				t.Log("nothing timely: slot would be skipped")
				return
			}
			t.Logf("subject: %s", cctx.Subject)
			assert.True(t, strings.HasPrefix(cctx.Subject, "community:"))
			assert.NotEmpty(t, cctx.URL)

			for _, p := range []social.Platform{social.LinkedIn, social.Bluesky, social.Mastodon} {
				out, err := c.Generate(ctx, nil, p, cctx)
				require.NoError(t, err)
				t.Logf("%s:\n%s", p, out)
				assert.NotContains(t, out, "{{", "unrendered placeholder")
				assert.NotContains(t, out, "—", "em dashes are off-brand")
				assert.Contains(t, out, cctx.URL)
				if p == social.Bluesky {
					assert.LessOrEqual(t, utf8.RuneCountInString(out), 300, "Bluesky limit")
				}
			}
		})
	}
}
