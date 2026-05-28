// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/env"
	mockai "github.com/ainsleyclark/godaily/pkg/mocks/ai"
	mockdigest "github.com/ainsleyclark/godaily/pkg/mocks/digest"
	mocknews "github.com/ainsleyclark/godaily/pkg/mocks/news"
	mockslack "github.com/ainsleyclark/godaily/pkg/mocks/slack"
	mocksocial "github.com/ainsleyclark/godaily/pkg/mocks/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/candidate"
	"github.com/ainsleyclark/godaily/pkg/services/social/platform"
)

// fakeCandidate is a hand-written test stub for the Candidate interface.
// Each test case constructs one with the eligibility outcome it needs;
// we don't gomock-generate the candidate because the surface is small
// and the literal-struct usage is clearer at the test site.
type fakeCandidate struct {
	kind     social.PostKind
	eligible bool
	ctx      candidate.CandidateContext
	err      error
	text     string
}

func (f *fakeCandidate) Kind() social.PostKind { return f.kind }

func (f *fakeCandidate) Eligible(_ context.Context, _ time.Time) (candidate.CandidateContext, bool, error) {
	if f.err != nil {
		return candidate.CandidateContext{}, false, f.err
	}
	if !f.eligible {
		return candidate.CandidateContext{}, false, nil
	}
	cctx := f.ctx
	cctx.Kind = f.kind
	return cctx, true, nil
}

func (f *fakeCandidate) Generate(_ context.Context, _ ai.Prompter, _ social.Platform, _ candidate.CandidateContext) (string, error) {
	return f.text, nil
}

// rotationFixture wires a Service for rotation tests. The slack sender
// accepts any call so candidate errors don't fail the test on Slack.
type rotationFixture struct {
	svc      *Service
	posts    *mocksocial.MockPostRepository
	prompter *mockai.MockPrompter
	poster   *mocksocial.MockPoster
}

func newRotationFixture(t *testing.T, cands ...candidate.Candidate) rotationFixture {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	slk := mockslack.NewMockSender(ctrl)
	slk.EXPECT().MustSend(gomock.Any(), gomock.Any()).AnyTimes()

	prompter := mockai.NewMockPrompter(ctrl)
	issues := mockdigest.NewMockIssueRepository(ctrl)
	items := mocknews.NewMockItemRepository(ctrl)
	posts := mocksocial.NewMockPostRepository(ctrl)

	bluesky := mocksocial.NewMockPoster(ctrl)
	bluesky.EXPECT().Platform().Return(social.Bluesky).AnyTimes()

	svc, err := New(env.Config{}, prompter, issues, items, posts, nil, slk)
	require.NoError(t, err)
	svc.posters = []platform.Poster{bluesky}
	svc.candidates = cands

	return rotationFixture{svc: svc, posts: posts, prompter: prompter, poster: bluesky}
}

var (
	// Calendar reference points for the day-routing tests. All at 15:00
	// UTC — the scheduled rotation time.
	tueAt15 = time.Date(2026, 5, 19, 15, 0, 0, 0, time.UTC) // Tuesday
	wedAt15 = time.Date(2026, 5, 20, 15, 0, 0, 0, time.UTC) // Wednesday (community)
	thuAt15 = time.Date(2026, 5, 21, 15, 0, 0, 0, time.UTC) // Thursday (no-op day)
	friAt15 = time.Date(2026, 5, 22, 15, 0, 0, 0, time.UTC) // Friday
)

func TestService_Rotate(t *testing.T) {
	t.Run("Tuesday dry-run picks first eligible candidate", func(t *testing.T) {
		newSrc := &fakeCandidate{kind: social.PostKindNewSource, eligible: false}
		spot := &fakeCandidate{
			kind:     social.PostKindSpotlight,
			eligible: true,
			ctx:      candidate.CandidateContext{Subject: "spotlight:ardanlabs"},
			text:     "Follow @ardanlabs for great Go content.",
		}
		cta := &fakeCandidate{kind: social.PostKindCTA, eligible: true}
		f := newRotationFixture(t, newSrc, spot, cta)

		res, err := f.svc.Rotate(context.Background(), social.RotateOptions{Now: tueAt15, DryRun: true})
		require.NoError(t, err)
		require.Len(t, res, 1)
		assert.Equal(t, social.PostKindSpotlight, res[0].Kind)
		assert.Equal(t, "Follow @ardanlabs for great Go content.", res[0].Text)
	})

	t.Run("Tuesday falls through to CTA when others ineligible", func(t *testing.T) {
		newSrc := &fakeCandidate{kind: social.PostKindNewSource, eligible: false}
		spot := &fakeCandidate{kind: social.PostKindSpotlight, eligible: false}
		cta := &fakeCandidate{
			kind:     social.PostKindCTA,
			eligible: true,
			ctx:      candidate.CandidateContext{Subject: "cta:angle-0"},
			text:     "Sign up to GoDaily.",
		}
		f := newRotationFixture(t, newSrc, spot, cta)

		res, err := f.svc.Rotate(context.Background(), social.RotateOptions{Now: tueAt15, DryRun: true})
		require.NoError(t, err)
		require.Len(t, res, 1)
		assert.Equal(t, social.PostKindCTA, res[0].Kind)
	})

	t.Run("Tuesday with no eligible candidates is a no-op", func(t *testing.T) {
		newSrc := &fakeCandidate{kind: social.PostKindNewSource, eligible: false}
		spot := &fakeCandidate{kind: social.PostKindSpotlight, eligible: false}
		cta := &fakeCandidate{kind: social.PostKindCTA, eligible: false}
		f := newRotationFixture(t, newSrc, spot, cta)

		res, err := f.svc.Rotate(context.Background(), social.RotateOptions{Now: tueAt15, DryRun: true})
		require.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("Friday ignores Tuesday candidates and no-ops without recap", func(t *testing.T) {
		// Even though all three Tuesday candidates are eligible, Friday
		// must ignore them. With no recap candidate registered, this is
		// a no-op.
		newSrc := &fakeCandidate{kind: social.PostKindNewSource, eligible: true, text: "x"}
		spot := &fakeCandidate{kind: social.PostKindSpotlight, eligible: true, text: "y"}
		cta := &fakeCandidate{kind: social.PostKindCTA, eligible: true, text: "z"}
		f := newRotationFixture(t, newSrc, spot, cta)

		res, err := f.svc.Rotate(context.Background(), social.RotateOptions{Now: friAt15, DryRun: true})
		require.NoError(t, err)
		assert.Empty(t, res, "Friday without recap registered must no-op")
	})

	t.Run("Friday runs recap when eligible", func(t *testing.T) {
		rec := &fakeCandidate{
			kind:     social.PostKindRecap,
			eligible: true,
			ctx:      candidate.CandidateContext{Subject: "recap:2026-W21"},
			text:     "Top stories this week …",
		}
		f := newRotationFixture(t, rec)

		res, err := f.svc.Rotate(context.Background(), social.RotateOptions{Now: friAt15, DryRun: true})
		require.NoError(t, err)
		require.Len(t, res, 1)
		assert.Equal(t, social.PostKindRecap, res[0].Kind)
	})

	t.Run("Non-rotation day is a no-op", func(t *testing.T) {
		always := &fakeCandidate{kind: social.PostKindNewSource, eligible: true, text: "x"}
		f := newRotationFixture(t, always)

		res, err := f.svc.Rotate(context.Background(), social.RotateOptions{Now: thuAt15, DryRun: true})
		require.NoError(t, err)
		assert.Empty(t, res, "Thursday is not a rotation day")
	})

	t.Run("Wednesday runs the community candidate", func(t *testing.T) {
		community := &fakeCandidate{
			kind:     social.PostKindCommunity,
			eligible: true,
			ctx:      candidate.CandidateContext{Subject: "community:golang-london:2026"},
			text:     "shout-out to Golang London",
		}
		f := newRotationFixture(t, community)

		res, err := f.svc.Rotate(context.Background(), social.RotateOptions{Now: wedAt15, DryRun: true})
		require.NoError(t, err)
		require.Len(t, res, 1)
		assert.Equal(t, social.PostKindCommunity, res[0].Kind)
	})

	t.Run("ForceKind bypasses day-of-week routing", func(t *testing.T) {
		cta := &fakeCandidate{
			kind:     social.PostKindCTA,
			eligible: true,
			ctx:      candidate.CandidateContext{Subject: "cta:forced"},
			text:     "subscribe please",
		}
		f := newRotationFixture(t, cta)

		res, err := f.svc.Rotate(context.Background(), social.RotateOptions{
			Now:       wedAt15, // not a rotation day, but ForceKind overrides
			DryRun:    true,
			ForceKind: social.PostKindCTA,
		})
		require.NoError(t, err)
		require.Len(t, res, 1)
		assert.Equal(t, social.PostKindCTA, res[0].Kind)
	})

	t.Run("Unknown ForceKind returns an error", func(t *testing.T) {
		f := newRotationFixture(t, &fakeCandidate{kind: social.PostKindCTA, eligible: true})

		_, err := f.svc.Rotate(context.Background(), social.RotateOptions{
			Now:       wedAt15,
			ForceKind: "no_such_kind",
		})
		require.Error(t, err)
	})

	t.Run("Eligibility error is propagated", func(t *testing.T) {
		broken := &fakeCandidate{
			kind: social.PostKindNewSource,
			err:  errors.New("db down"),
		}
		f := newRotationFixture(t, broken)

		_, err := f.svc.Rotate(context.Background(), social.RotateOptions{
			Now:       tueAt15,
			ForceKind: social.PostKindNewSource,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db down")
	})

	t.Run("No posters configured is a no-op", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)

		slk := mockslack.NewMockSender(ctrl)
		prompter := mockai.NewMockPrompter(ctrl)
		issues := mockdigest.NewMockIssueRepository(ctrl)
		items := mocknews.NewMockItemRepository(ctrl)
		posts := mocksocial.NewMockPostRepository(ctrl)

		svc, err := New(env.Config{}, prompter, issues, items, posts, nil, slk)
		require.NoError(t, err)
		svc.candidates = []candidate.Candidate{&fakeCandidate{kind: social.PostKindCTA, eligible: true}}

		res, err := svc.Rotate(context.Background(), social.RotateOptions{Now: tueAt15})
		require.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("Wet run records the social post with kind and subject", func(t *testing.T) {
		spot := &fakeCandidate{
			kind:     social.PostKindSpotlight,
			eligible: true,
			ctx:      candidate.CandidateContext{Subject: "spotlight:ardanlabs"},
			text:     "Follow @ardanlabs for great Go content.",
		}
		f := newRotationFixture(t, spot)

		f.posts.EXPECT().
			HasPostedBySubject(gomock.Any(), "spotlight:ardanlabs", "bluesky").
			Return(false, nil)
		f.poster.EXPECT().
			Post(gomock.Any(), platform.PostRequest{Text: "Follow @ardanlabs for great Go content."}).
			Return(platform.PostResponse{PostURL: "https://bsky.app/x"}, nil)
		f.posts.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, p social.Post) (social.Post, error) {
				assert.Equal(t, social.PostKindSpotlight, p.Kind)
				assert.Equal(t, "spotlight:ardanlabs", p.Subject)
				assert.Nil(t, p.IssueID, "rotation rows must not carry an issue_id")
				assert.Equal(t, "https://bsky.app/x", p.PostURL)
				p.ID = 1
				return p, nil
			})

		res, err := f.svc.Rotate(context.Background(), social.RotateOptions{Now: tueAt15})
		require.NoError(t, err)
		require.Len(t, res, 1)
		assert.Equal(t, social.PostKindSpotlight, res[0].Kind)
		assert.Equal(t, "https://bsky.app/x", res[0].PostURL)
	})

	t.Run("Wet run skips when subject already posted", func(t *testing.T) {
		spot := &fakeCandidate{
			kind:     social.PostKindSpotlight,
			eligible: true,
			ctx:      candidate.CandidateContext{Subject: "spotlight:ardanlabs"},
			text:     "x",
		}
		f := newRotationFixture(t, spot)

		f.posts.EXPECT().
			HasPostedBySubject(gomock.Any(), "spotlight:ardanlabs", "bluesky").
			Return(true, nil)
		// No Post(), no Create() — the platform is skipped.

		res, err := f.svc.Rotate(context.Background(), social.RotateOptions{Now: tueAt15})
		require.NoError(t, err)
		require.Len(t, res, 1)
		assert.True(t, res[0].Skipped, "platform should report Skipped when already posted")
	})
}
