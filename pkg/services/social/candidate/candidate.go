// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package candidate defines the rotation Candidate interface and the
// CandidateContext value passed between Eligible and Generate. It lives
// in its own package so concrete candidate implementations can depend on
// the interface without import-cycling against pkg/services/social.
package candidate

import (
	"context"
	"time"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
)

// CandidateContext is the unit of work the rotation pipeline hands from
// a Candidate's Eligible check to the per-platform Generate step.
// Different candidate kinds populate different fields:
//
//   - new_source/recap/spotlight/cta all set Subject for idempotency.
//   - spotlight + new_source set Mentions (per-platform handle).
//   - recap puts the recap.Top into Payload so its generator can
//     render it without a second DB hit.
type CandidateContext struct {
	Kind     social.PostKind
	Hook     string
	URL      string
	Subject  string
	Mentions map[social.Platform]string
	Payload  any

	// LinkedInOrgURN is the LinkedIn organisation URN this candidate
	// represents, when one is known. The publish loop turns it into an
	// inline @-organisation annotation on the LinkedIn post, anchored
	// to LinkedInDisplayName. Both must be set for the annotation to
	// render; either empty falls back to plain-text (current behaviour).
	LinkedInOrgURN      string
	LinkedInDisplayName string
}

// Candidate is one possible rotation post. Eligible looks at the world
// (DB, click metrics) and either returns a populated context or reports
// the candidate is not ready.
type Candidate interface {
	// Kind reports the SocialPostKind this candidate produces.
	Kind() social.PostKind
	// Eligible reports whether the candidate can post right now. The
	// bool is the source of truth; the context is only meaningful when
	// true.
	Eligible(ctx context.Context, now time.Time) (CandidateContext, bool, error)
	// Generate returns the post text for one platform given the context
	// from Eligible.
	Generate(ctx context.Context, p ai.Prompter, platform social.Platform, c CandidateContext) (string, error)
}
