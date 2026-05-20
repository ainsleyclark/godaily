// Copyright (c) 2026 godaily (Ainsley Clark)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package mastodon

import (
	"context"
	"errors"
	"testing"

	"github.com/mattn/go-mastodon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/pkg/gateway/social"
)

func TestClient_Platform(t *testing.T) {
	t.Parallel()

	c := New("https://mastodon.social", "tok")
	assert.Equal(t, social.PlatformMastodon, c.Platform())
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
