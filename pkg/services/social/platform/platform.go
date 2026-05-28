// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package platform

import (
	"context"

	"github.com/ainsleyclark/godaily/pkg/domain/social"
)

//go:generate go run go.uber.org/mock/mockgen -package=mocksocial -destination=../../../mocks/social/Poster.go github.com/ainsleyclark/godaily/pkg/services/social/platform Poster

// Poster publishes a single text post on one social platform.
//
// Implementations must be safe to call from a serverless handler:
// no background goroutines that outlive the request, and any HTTP
// transport must respect the supplied context for cancellation.
type Poster interface {
	// Platform identifies which platform this poster targets.
	Platform() social.Platform

	// Post publishes a request to the platform. Implementations are
	// responsible for any auth dance their API requires.
	Post(ctx context.Context, req PostRequest) (PostResponse, error)
}

// PostRequest is the per-post payload handed to a Poster. Text is the
// fully-rendered post body. Mentions carries every mentionable identity
// the publish loop wants attached to this post — implementations filter
// by m.Platform and decide how to render them (LinkedIn builds inline
// annotations; Bluesky / Mastodon ignore the list because their handles
// are already inlined in Text by the prompt layer).
type PostRequest struct {
	Text     string
	Mentions []social.Mention
}

// PostResponse is what a platform returned after a successful post.
// PostURL is the canonical web URL of the published content when
// available — implementations leave it empty when the platform does not
// return one synchronously.
type PostResponse struct {
	PostURL string
}
