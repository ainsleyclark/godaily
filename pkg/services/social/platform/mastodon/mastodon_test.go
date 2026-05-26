// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mastodon

import (
	"context"
	"errors"
	"testing"

	"github.com/mattn/go-mastodon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/pkg/domain/social"
)

func TestClient_Platform(t *testing.T) {
	t.Parallel()

	c := New("https://mastodon.social", "tok")
	assert.Equal(t, social.Mastodon, c.Platform())
}

func TestClient_Post(t *testing.T) {
	t.Parallel()

	t.Run("Happy path returns status URL", func(t *testing.T) {
		t.Parallel()

		c := &Client{
			postStatusFunc: func(_ context.Context, toot *mastodon.Toot) (*mastodon.Status, error) {
				assert.Equal(t, "Hello, fediverse!", toot.Status)
				assert.Equal(t, "public", toot.Visibility)
				return &mastodon.Status{URL: "https://mastodon.social/@godaily/123"}, nil
			},
		}

		got, err := c.Post(context.Background(), "Hello, fediverse!")
		require.NoError(t, err)
		assert.Equal(t, "https://mastodon.social/@godaily/123", got.PostURL)
	})

	t.Run("Nil status from SDK yields empty URL", func(t *testing.T) {
		t.Parallel()

		c := &Client{
			postStatusFunc: func(_ context.Context, _ *mastodon.Toot) (*mastodon.Status, error) {
				return nil, nil
			},
		}

		got, err := c.Post(context.Background(), "x")
		require.NoError(t, err)
		assert.Empty(t, got.PostURL)
	})

	t.Run("SDK error wrapped", func(t *testing.T) {
		t.Parallel()

		c := &Client{
			postStatusFunc: func(_ context.Context, _ *mastodon.Toot) (*mastodon.Status, error) {
				return nil, errors.New("network down")
			},
		}

		_, err := c.Post(context.Background(), "x")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "mastodon PostStatus")
		assert.Contains(t, err.Error(), "network down")
	})
}
