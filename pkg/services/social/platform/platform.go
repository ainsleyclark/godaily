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

	// Post publishes text to the platform. Implementations are responsible
	// for any auth dance their API requires.
	Post(ctx context.Context, text string) (Result, error)
}

// Result is what a platform returned after a successful post. PostURL is the
// canonical web URL of the published content when available — implementations
// leave it empty when the platform does not return one synchronously.
type Result struct {
	PostURL string
}
