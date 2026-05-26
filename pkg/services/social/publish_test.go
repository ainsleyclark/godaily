// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"testing"

	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/platform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSelectPosters(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	bluesky := newMockPoster(ctrl, social.Bluesky)
	linkedin := newMockPoster(ctrl, social.LinkedIn)
	mastodon := newMockPoster(ctrl, social.Mastodon)
	all := []platform.Poster{bluesky, linkedin, mastodon}

	t.Run("Empty wanted returns all", func(t *testing.T) {
		t.Parallel()
		got := selectPosters(all, nil)
		assert.Len(t, got, 3)
	})

	t.Run("Subset returns only matches", func(t *testing.T) {
		t.Parallel()
		got := selectPosters(all, []social.Platform{social.LinkedIn, social.Mastodon})
		require.Len(t, got, 2)
		assert.Equal(t, social.LinkedIn, got[0].Platform())
		assert.Equal(t, social.Mastodon, got[1].Platform())
	})

	t.Run("Unknown wanted yields empty", func(t *testing.T) {
		t.Parallel()
		got := selectPosters(all, []social.Platform{"twitter"})
		assert.Empty(t, got)
	})
}
