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

package social_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	socialgw "github.com/ainsleyclark/godaily/pkg/gateway/social"
	mockai "github.com/ainsleyclark/godaily/pkg/mocks/ai"
	mocknews "github.com/ainsleyclark/godaily/pkg/mocks/domain/news"
	mockslack "github.com/ainsleyclark/godaily/pkg/mocks/slack"
	mocksocial "github.com/ainsleyclark/godaily/pkg/mocks/social"
	"github.com/ainsleyclark/godaily/pkg/services/social"
)

// fakeCandidate is a stand-in Candidate that drops in any eligibility
// outcome the test needs. Reused across the day-routing tests below.
type fakeCandidate struct {
	kind     news.SocialPostKind
	eligible bool
	ctx      social.CandidateContext
	err      error
	text     string
}

func (f *fakeCandidate) Kind() news.SocialPostKind { return f.kind }

func (f *fakeCandidate) Eligible(_ context.Context, _ time.Time) (social.CandidateContext, bool, error) {
	if f.err != nil {
		return social.CandidateContext{}, false, f.err
	}
	if !f.eligible {
		return social.CandidateContext{}, false, nil
	}
	cctx := f.ctx
	cctx.Kind = f.kind
	return cctx, true, nil
}

func (f *fakeCandidate) Generate(_ context.Context, _ ai.Prompter, _ socialgw.Platform, _ social.CandidateContext) (string, error) {
	return f.text, nil
}

// rotationFixture wires a Service for rotation tests. The slack sender
// accepts any call so candidate errors don't fail the test on Slack.
type rotationFixture struct {
	svc      *social.Service
	posts    *mocknews.MockSocialPostRepository
	prompter *mockai.MockPrompter
	poster   *mocksocial.MockPoster
}

func newRotationFixture(t *testing.T, candidates ...social.Candidate) rotationFixture {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	slk := mockslack.NewMockSender(ctrl)
	slk.EXPECT().MustSend(gomock.Any(), gomock.Any()).AnyTimes()

	prompter := mockai.NewMockPrompter(ctrl)
	issues := mocknews.NewMockIssueRepository(ctrl)
	items := mocknews.NewMockItemRepository(ctrl)
	posts := mocknews.NewMockSocialPostRepository(ctrl)

	bluesky := mocksocial.NewMockPoster(ctrl)
	bluesky.EXPECT().Platform().Return(socialgw.PlatformBluesky).AnyTimes()

	svc, err := social.New([]socialgw.Poster{bluesky}, prompter, issues, items, posts, slk)
	require.NoError(t, err)
	svc.WithCandidates(candidates...)

	return rotationFixture{svc: svc, posts: posts, prompter: prompter, poster: bluesky}
}

var (
	// Tuesday and Friday calendar reference points for the day-routing
	// tests. Both at 15:00 UTC — the scheduled rotation time.
	tueAt15 = time.Date(2026, 5, 19, 15, 0, 0, 0, time.UTC) // Tuesday
	friAt15 = time.Date(2026, 5, 22, 15, 0, 0, 0, time.UTC) // Friday
	wedAt15 = time.Date(2026, 5, 20, 15, 0, 0, 0, time.UTC) // Wednesday (no-op day)
)

func TestRotate_WetRunRecordsSocialPostWithKindAndSubject(t *testing.T) {
	spot := &fakeCandidate{
		kind:     news.SocialPostKindSpotlight,
		eligible: true,
		ctx:      social.CandidateContext{Subject: "spotlight:ardanlabs"},
		text:     "Follow @ardanlabs for great Go content.",
	}
	f := newRotationFixture(t, spot)

	// publish() probes HasPostedBySubject before calling the poster.
	f.posts.EXPECT().
		HasPostedBySubject(gomock.Any(), "spotlight:ardanlabs", "bluesky").
		Return(false, nil)

	// Then the poster publishes …
	f.poster.EXPECT().
		Post(gomock.Any(), "Follow @ardanlabs for great Go content.").
		Return(socialgw.Result{PostURL: "https://bsky.app/x"}, nil)

	// … and the result is recorded with the right fields.
	f.posts.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p news.SocialPost) (news.SocialPost, error) {
			assert.Equal(t, news.SocialPostKindSpotlight, p.Kind)
			assert.Equal(t, "spotlight:ardanlabs", p.Subject)
			assert.Nil(t, p.IssueID, "rotation rows must not carry an issue_id")
			assert.Equal(t, "https://bsky.app/x", p.PostURL)
			p.ID = 1
			return p, nil
		})

	res, err := f.svc.Rotate(context.Background(), social.RotateOptions{Now: tueAt15})
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, news.SocialPostKindSpotlight, res[0].Kind)
	assert.Equal(t, "https://bsky.app/x", res[0].PostURL)
}

func TestRotate_WetRunSkipsWhenSubjectAlreadyPosted(t *testing.T) {
	spot := &fakeCandidate{
		kind:     news.SocialPostKindSpotlight,
		eligible: true,
		ctx:      social.CandidateContext{Subject: "spotlight:ardanlabs"},
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
}

func TestRotate_TuesdayDryRunPicksSpotlight(t *testing.T) {
	selfRel := &fakeCandidate{kind: news.SocialPostKindSelfRelease, eligible: false}
	spot := &fakeCandidate{
		kind:     news.SocialPostKindSpotlight,
		eligible: true,
		ctx:      social.CandidateContext{Subject: "spotlight:ardanlabs"},
		text:     "Follow @ardanlabs for great Go content.",
	}
	cta := &fakeCandidate{kind: news.SocialPostKindCTA, eligible: true}

	f := newRotationFixture(t, selfRel, spot, cta)

	res, err := f.svc.Rotate(context.Background(), social.RotateOptions{
		Now:    tueAt15,
		DryRun: true,
	})
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, news.SocialPostKindSpotlight, res[0].Kind)
	assert.Equal(t, "Follow @ardanlabs for great Go content.", res[0].Text)
}

func TestRotate_TuesdayFallsThroughToCTAWhenOthersNotEligible(t *testing.T) {
	selfRel := &fakeCandidate{kind: news.SocialPostKindSelfRelease, eligible: false}
	spot := &fakeCandidate{kind: news.SocialPostKindSpotlight, eligible: false}
	cta := &fakeCandidate{
		kind:     news.SocialPostKindCTA,
		eligible: true,
		ctx:      social.CandidateContext{Subject: "cta:angle-0"},
		text:     "Sign up to GoDaily.",
	}

	f := newRotationFixture(t, selfRel, spot, cta)

	res, err := f.svc.Rotate(context.Background(), social.RotateOptions{Now: tueAt15, DryRun: true})
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, news.SocialPostKindCTA, res[0].Kind)
}

func TestRotate_TuesdayNoEligibleCandidatesIsNoOp(t *testing.T) {
	selfRel := &fakeCandidate{kind: news.SocialPostKindSelfRelease, eligible: false}
	spot := &fakeCandidate{kind: news.SocialPostKindSpotlight, eligible: false}
	cta := &fakeCandidate{kind: news.SocialPostKindCTA, eligible: false}
	f := newRotationFixture(t, selfRel, spot, cta)

	res, err := f.svc.Rotate(context.Background(), social.RotateOptions{Now: tueAt15, DryRun: true})
	require.NoError(t, err)
	assert.Empty(t, res)
}

func TestRotate_FridayOnlyConsidersRecap(t *testing.T) {
	// Even though all three Tuesday candidates are eligible, Friday must
	// ignore them. With no recap candidate registered, this is a no-op.
	selfRel := &fakeCandidate{kind: news.SocialPostKindSelfRelease, eligible: true, text: "x"}
	spot := &fakeCandidate{kind: news.SocialPostKindSpotlight, eligible: true, text: "y"}
	cta := &fakeCandidate{kind: news.SocialPostKindCTA, eligible: true, text: "z"}
	f := newRotationFixture(t, selfRel, spot, cta)

	res, err := f.svc.Rotate(context.Background(), social.RotateOptions{Now: friAt15, DryRun: true})
	require.NoError(t, err)
	assert.Empty(t, res, "Friday without recap registered must no-op")
}

func TestRotate_FridayRunsRecapWhenEligible(t *testing.T) {
	recap := &fakeCandidate{
		kind:     news.SocialPostKindRecap,
		eligible: true,
		ctx:      social.CandidateContext{Subject: "recap:2026-W21"},
		text:     "Top stories this week …",
	}
	f := newRotationFixture(t, recap)

	res, err := f.svc.Rotate(context.Background(), social.RotateOptions{Now: friAt15, DryRun: true})
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, news.SocialPostKindRecap, res[0].Kind)
}

func TestRotate_OffDayIsNoOp(t *testing.T) {
	always := &fakeCandidate{kind: news.SocialPostKindSelfRelease, eligible: true, text: "x"}
	f := newRotationFixture(t, always)

	res, err := f.svc.Rotate(context.Background(), social.RotateOptions{Now: wedAt15, DryRun: true})
	require.NoError(t, err)
	assert.Empty(t, res, "Wednesday is not a rotation day")
}

func TestRotate_ForceKindBypassesDayRouting(t *testing.T) {
	cta := &fakeCandidate{
		kind:     news.SocialPostKindCTA,
		eligible: true,
		ctx:      social.CandidateContext{Subject: "cta:forced"},
		text:     "subscribe please",
	}
	f := newRotationFixture(t, cta)

	res, err := f.svc.Rotate(context.Background(), social.RotateOptions{
		Now:       wedAt15, // not a rotation day, but ForceKind overrides
		DryRun:    true,
		ForceKind: news.SocialPostKindCTA,
	})
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, news.SocialPostKindCTA, res[0].Kind)
}

func TestRotate_ForceKindUnknownIsError(t *testing.T) {
	f := newRotationFixture(t, &fakeCandidate{kind: news.SocialPostKindCTA, eligible: true})

	_, err := f.svc.Rotate(context.Background(), social.RotateOptions{
		Now:       wedAt15,
		ForceKind: "no_such_kind",
	})
	require.Error(t, err)
}

func TestRotate_EligibilityErrorIsPropagated(t *testing.T) {
	broken := &fakeCandidate{
		kind: news.SocialPostKindSelfRelease,
		err:  errors.New("github down"),
	}
	f := newRotationFixture(t, broken)

	_, err := f.svc.Rotate(context.Background(), social.RotateOptions{
		Now:       tueAt15,
		ForceKind: news.SocialPostKindSelfRelease,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "github down")
}

func TestRotate_NoPostersIsNoOp(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	slk := mockslack.NewMockSender(ctrl)
	prompter := mockai.NewMockPrompter(ctrl)
	issues := mocknews.NewMockIssueRepository(ctrl)
	items := mocknews.NewMockItemRepository(ctrl)
	posts := mocknews.NewMockSocialPostRepository(ctrl)

	svc, err := social.New(nil, prompter, issues, items, posts, slk)
	require.NoError(t, err)
	svc.WithCandidates(&fakeCandidate{kind: news.SocialPostKindCTA, eligible: true})

	res, err := svc.Rotate(context.Background(), social.RotateOptions{Now: tueAt15})
	require.NoError(t, err)
	assert.Empty(t, res)
}
