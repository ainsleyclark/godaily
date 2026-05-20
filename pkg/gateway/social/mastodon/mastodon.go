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

// Package mastodon publishes statuses to a Mastodon instance via the
// github.com/mattn/go-mastodon SDK.
package mastodon

import (
	"context"

	"github.com/mattn/go-mastodon"
	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/gateway/social"
)

// postStatusFn matches mastodon.Client.PostStatus so tests can stub the
// network without depending on the package's concrete type.
type postStatusFn func(ctx context.Context, toot *mastodon.Toot) (*mastodon.Status, error)

// Client publishes statuses to a Mastodon instance.
type Client struct {
	postStatusFunc postStatusFn
}

// New creates a new Mastodon Client. server is the full base URL of the
// instance (e.g. "https://mastodon.social"); accessToken is an app-token
// obtained from the user's Preferences → Development page with at least
// the "write:statuses" scope.
func New(server, accessToken string) *Client {
	c := mastodon.NewClient(&mastodon.Config{
		Server:      server,
		AccessToken: accessToken,
	})
	return &Client{postStatusFunc: c.PostStatus}
}

// Platform implements social.Poster.
func (c *Client) Platform() social.Platform {
	return social.PlatformMastodon
}

// Post publishes text as a public status on the configured instance.
func (c *Client) Post(ctx context.Context, text string) (social.Result, error) {
	status, err := c.postStatusFunc(ctx, &mastodon.Toot{
		Status:     text,
		Visibility: "public",
	})
	if err != nil {
		return social.Result{}, errors.Wrap(err, "mastodon PostStatus")
	}
	if status == nil {
		return social.Result{}, nil
	}
	return social.Result{PostURL: status.URL}, nil
}
