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

package social

import "context"

// Stats holds the engagement counts returned by a platform for a single post.
type Stats struct {
	Likes       int64
	Reposts     int64
	Comments    int64
	Impressions int64
}

//go:generate go run go.uber.org/mock/mockgen -package=mocksocial -destination=../../mocks/social/StatFetcher.go github.com/ainsleyclark/godaily/pkg/gateway/social StatFetcher

// StatFetcher fetches engagement stats for a published post by its URL.
type StatFetcher interface {
	// Platform identifies the platform this fetcher targets.
	Platform() Platform

	// GetStats returns the current engagement counts for the post at postURL.
	// postURL is the canonical web URL stored in social_posts.post_url.
	GetStats(ctx context.Context, postURL string) (Stats, error)
}
