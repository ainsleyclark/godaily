// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"context"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/candidate"
)

// pickCandidates returns the rotation candidate list for the day, or
// nil when the day is not a rotation day. The DraftAll path walks the
// returned slice and stops at the first eligible candidate.
//
// Day-of-week rules:
//
//   - Monday: recap (the previous week's top-clicks). Moved off Friday
//     so the metric window covers a complete Mon–Fri + weekend.
//   - Wednesday: community (a Go conference or meetup), 2:1 meetup:conf
//     rotation through the curated lists in pkg/data.
//   - Friday: new_source → spotlight → cta → no-op.
//   - Other days: no-op.
func (s *Service) pickCandidates(weekday time.Weekday) []candidate.Candidate {
	switch weekday {
	case time.Monday:
		return orderedByKinds(s.candidates, social.PostKindRecap)
	case time.Wednesday:
		return orderedByKinds(s.candidates, social.PostKindCommunity)
	case time.Friday:
		return orderedByKinds(
			s.candidates,
			social.PostKindNewSource,
			social.PostKindSpotlight,
			social.PostKindCTA,
		)
	default:
		return nil
	}
}

// orderedByKinds returns the subset of candidates matching the given
// kinds, in the requested order. Missing candidates are silently dropped
// — useful when a deployment hasn't wired every kind.
func orderedByKinds(all []candidate.Candidate, kinds ...social.PostKind) []candidate.Candidate {
	out := make([]candidate.Candidate, 0, len(kinds))
	for _, k := range kinds {
		if c := candidateByKind(all, k); c != nil {
			out = append(out, c)
		}
	}
	return out
}

// subjectIdempotency returns a skipIfPosted check keyed off the
// candidate's Subject. An empty subject disables the check (caller is
// trusting the candidate's own eligibility logic). Cancelled rows count
// as "already handled" so a deliberately-skipped draft is not
// regenerated on the next build.
func subjectIdempotency(posts social.PostRepository, subject string) func(ctx context.Context, platform string) (bool, error) {
	if subject == "" {
		return nil
	}
	return func(ctx context.Context, platform string) (bool, error) {
		return posts.HasPostedOrCancelledBySubject(ctx, subject, platform)
	}
}
