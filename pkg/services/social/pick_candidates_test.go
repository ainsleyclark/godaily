// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/candidate"
)

// fakeCandidate is a hand-written stub for the Candidate interface, used
// to feed pickCandidates a known kind without bootstrapping a real one.
type fakeCandidate struct {
	kind social.PostKind
}

func (f *fakeCandidate) Kind() social.PostKind { return f.kind }

func (f *fakeCandidate) Eligible(_ context.Context, _ time.Time) (candidate.CandidateContext, bool, error) {
	return candidate.CandidateContext{Kind: f.kind}, true, nil
}

func (f *fakeCandidate) Generate(_ context.Context, _ ai.Prompter, _ social.Platform, _ candidate.CandidateContext) (string, error) {
	return "", nil
}

// Compile-time check that fakeCandidate satisfies the interface.
var _ candidate.Candidate = (*fakeCandidate)(nil)

func TestPickCandidates(t *testing.T) {
	t.Parallel()

	all := []candidate.Candidate{
		&fakeCandidate{kind: social.PostKindNewSource},
		&fakeCandidate{kind: social.PostKindSpotlight},
		&fakeCandidate{kind: social.PostKindCTA},
		&fakeCandidate{kind: social.PostKindCommunity},
		&fakeCandidate{kind: social.PostKindRecap},
	}
	svc := &Service{candidates: all}

	t.Run("Monday returns recap", func(t *testing.T) {
		t.Parallel()
		got := svc.pickCandidates(time.Monday)
		assertKinds(t, got, social.PostKindRecap)
	})

	t.Run("Wednesday returns community", func(t *testing.T) {
		t.Parallel()
		got := svc.pickCandidates(time.Wednesday)
		assertKinds(t, got, social.PostKindCommunity)
	})

	t.Run("Friday returns new_source / spotlight / cta in order", func(t *testing.T) {
		t.Parallel()
		got := svc.pickCandidates(time.Friday)
		assertKinds(t, got, social.PostKindNewSource, social.PostKindSpotlight, social.PostKindCTA)
	})

	for _, day := range []time.Weekday{time.Sunday, time.Tuesday, time.Thursday, time.Saturday} {
		t.Run("No-op day: "+day.String(), func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, svc.pickCandidates(day))
		})
	}

	t.Run("Missing candidates are silently dropped", func(t *testing.T) {
		t.Parallel()
		// A service wired with only the recap candidate must still return
		// a (single-element) slice on Monday and an empty slice on Friday.
		monRecapOnly := &Service{candidates: []candidate.Candidate{&fakeCandidate{kind: social.PostKindRecap}}}
		assertKinds(t, monRecapOnly.pickCandidates(time.Monday), social.PostKindRecap)
		assert.Empty(t, monRecapOnly.pickCandidates(time.Friday))
	})
}

func assertKinds(t *testing.T, got []candidate.Candidate, want ...social.PostKind) {
	t.Helper()
	gotKinds := make([]social.PostKind, len(got))
	for i, c := range got {
		gotKinds[i] = c.Kind()
	}
	assert.Equal(t, want, gotKinds)
}
