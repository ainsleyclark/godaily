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

import (
	"context"
	"time"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/gateway/social"
)

// CandidateContext is the unit of work the rotation pipeline hands from a
// Candidate's Eligible check to the per-platform Generate step. Different
// candidate kinds populate different fields:
//
//   - self_release/recap/spotlight/cta all set Subject for idempotency.
//   - spotlight sets Mentions (per-platform handle, already formatted).
//   - recap puts the recap.Top into Payload so its generator can render it
//     without a second DB hit.
type CandidateContext struct {
	Kind     news.SocialPostKind
	Hook     string
	URL      string
	Subject  string
	Mentions map[social.Platform]string
	Payload  any
}

// Generator produces post text for one platform from a CandidateContext.
// Returning a non-nil error aborts the publish for that platform only.
type Generator func(ctx context.Context, p ai.Prompter, platform social.Platform, c CandidateContext) (string, error)

// Candidate is one possible rotation post. Eligible looks at the world
// (DB, GitHub, click metrics) and either returns a populated context or
// reports the candidate is not ready.
type Candidate interface {
	// Kind reports the SocialPostKind this candidate produces.
	Kind() news.SocialPostKind

	// Eligible reports whether the candidate can post right now. The bool
	// is the source of truth; the context is only meaningful when true.
	Eligible(ctx context.Context, now time.Time) (CandidateContext, bool, error)

	// Generate returns the post text for one platform given the context
	// from Eligible.
	Generate(ctx context.Context, p ai.Prompter, platform social.Platform, c CandidateContext) (string, error)
}

// CandidateByKind returns the candidate with the given Kind, or nil if
// none is registered. Used by the CLI to drive a single kind for testing.
func CandidateByKind(all []Candidate, kind news.SocialPostKind) Candidate {
	for _, c := range all {
		if c.Kind() == kind {
			return c
		}
	}
	return nil
}
