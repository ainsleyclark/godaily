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
	"fmt"
	"net/url"
	"strings"

	"github.com/mattn/go-mastodon"
	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/services/social/platform"
)

// Client publishes statuses to a Mastodon instance.
type Client struct {
	postStatusFunc postStatusFn
	getStatusFunc  getStatusFn
}

// postStatusFn matches mastodon.Client.PostStatus so tests can stub the
// network without depending on the package's concrete type.
type postStatusFn func(ctx context.Context, toot *mastodon.Toot) (*mastodon.Status, error)

// getStatusFn matches mastodon.Client.GetStatus for the same reason.
type getStatusFn func(ctx context.Context, id mastodon.ID) (*mastodon.Status, error)

// New creates a new Mastodon Client. server is the full base URL of the
// instance (e.g. "https://mastodon.social"); accessToken is an app-token
// obtained from the user's Preferences → Development page with at least
// the "write:statuses" scope.
func New(server, accessToken string) *Client {
	c := mastodon.NewClient(&mastodon.Config{
		Server:      server,
		AccessToken: accessToken,
	})
	return &Client{
		postStatusFunc: c.PostStatus,
		getStatusFunc:  c.GetStatus,
	}
}

// Platform implements platform.Poster.
func (c *Client) Platform() platform.Name {
	return platform.Mastodon
}

// Stats fetches engagement counts for a Mastodon status. postURL must be
// the canonical status URL (e.g. https://mastodon.social/@handle/113456789).
// The status ID is extracted from the URL path's last segment.
func (c *Client) Stats(ctx context.Context, postURL string) (platform.Stats, error) {
	id, err := statusIDFromURL(postURL)
	if err != nil {
		return platform.Stats{}, errors.Wrap(err, "extracting status ID from post URL")
	}
	status, err := c.getStatusFunc(ctx, id)
	if err != nil {
		return platform.Stats{}, errors.Wrap(err, "mastodon GetStatus")
	}
	if status == nil {
		return platform.Stats{}, nil
	}
	return platform.Stats{
		Likes:    status.FavouritesCount,
		Reposts:  status.ReblogsCount,
		Comments: status.RepliesCount,
	}, nil
}

// statusIDFromURL extracts the Mastodon status ID from a status URL.
// URL form: https://{instance}/@{handle}/{id}
func statusIDFromURL(postURL string) (mastodon.ID, error) {
	u, err := url.Parse(postURL)
	if err != nil {
		return "", errors.Wrap(err, "parsing post URL")
	}
	parts := strings.Split(strings.TrimRight(u.Path, "/"), "/")
	if len(parts) == 0 {
		return "", fmt.Errorf("cannot extract ID from Mastodon URL: %s", postURL)
	}
	id := parts[len(parts)-1]
	if id == "" {
		return "", fmt.Errorf("empty ID in Mastodon URL: %s", postURL)
	}
	return mastodon.ID(id), nil
}

// Post publishes text as a public status on the configured instance.
func (c *Client) Post(ctx context.Context, text string) (platform.Result, error) {
	status, err := c.postStatusFunc(ctx, &mastodon.Toot{
		Status:     text,
		Visibility: "public",
	})
	if err != nil {
		return platform.Result{}, errors.Wrap(err, "mastodon PostStatus")
	}

	if status == nil {
		return platform.Result{}, nil
	}

	return platform.Result{PostURL: status.URL}, nil
}
