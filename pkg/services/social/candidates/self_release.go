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

package candidates

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	socialgw "github.com/ainsleyclark/godaily/pkg/gateway/social"
	socialsvc "github.com/ainsleyclark/godaily/pkg/services/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/prompts/rotation"
)

// GitHubRelease is the subset of fields the rotation candidate consumes.
// Implementations live in pkg/gateway (HTTP) or pkg/mocks (tests).
type GitHubRelease struct {
	TagName     string
	Name        string
	HTMLURL     string
	Body        string
	PublishedAt time.Time
	Prerelease  bool
	Draft       bool
}

// ReleaseFetcher pulls the latest releases for a repo. Kept narrow so a
// test or alternate implementation can satisfy it without bringing in
// the HTTP gateway.
type ReleaseFetcher interface {
	LatestReleases(ctx context.Context) ([]GitHubRelease, error)
}

// SelfRelease announces GoDaily's own GitHub releases. Idempotency key is
// "self_release:<tag>". Pre-releases and drafts are ignored.
type SelfRelease struct {
	fetcher ReleaseFetcher
	posts   news.SocialPostRepository
}

// NewSelfRelease constructs the candidate.
func NewSelfRelease(fetcher ReleaseFetcher, posts news.SocialPostRepository) *SelfRelease {
	return &SelfRelease{fetcher: fetcher, posts: posts}
}

// Kind reports the candidate's SocialPostKind.
func (c *SelfRelease) Kind() news.SocialPostKind { return news.SocialPostKindSelfRelease }

// Eligible picks the newest non-pre-release whose tag we haven't posted
// about on at least one platform. We probe with the "bluesky" platform
// (any platform works here — the goal is "have we said anything about
// this release anywhere?") so the candidate trips even if just one
// platform was wired up at the time.
func (c *SelfRelease) Eligible(ctx context.Context, _ time.Time) (socialsvc.CandidateContext, bool, error) {
	if c.fetcher == nil {
		return socialsvc.CandidateContext{}, false, nil
	}

	releases, err := c.fetcher.LatestReleases(ctx)
	if err != nil {
		return socialsvc.CandidateContext{}, false, errors.Wrap(err, "fetching GoDaily releases")
	}

	for _, rel := range releases {
		if rel.Draft || rel.Prerelease || rel.TagName == "" {
			continue
		}
		subject := "self_release:" + rel.TagName
		posted, err := c.posts.HasPostedBySubject(ctx, subject, "bluesky")
		if err != nil {
			return socialsvc.CandidateContext{}, false, errors.Wrap(err, "checking HasPostedBySubject")
		}
		if posted {
			continue
		}
		return socialsvc.CandidateContext{
			Kind:    c.Kind(),
			Subject: subject,
			URL:     rel.HTMLURL,
			Payload: rotation.SelfReleasePayload{
				Tag:         rel.TagName,
				Name:        rel.Name,
				URL:         rel.HTMLURL,
				Body:        truncate(rel.Body, 800),
				PublishedAt: rel.PublishedAt.UTC().Format(time.RFC3339),
			},
		}, true, nil
	}

	return socialsvc.CandidateContext{}, false, nil
}

// Generate hands the release payload to the rotation/self_release prompt.
func (c *SelfRelease) Generate(ctx context.Context, p ai.Prompter, platform socialgw.Platform, cctx socialsvc.CandidateContext) (string, error) {
	payload, ok := cctx.Payload.(rotation.SelfReleasePayload)
	if !ok {
		return "", errors.New("self_release: payload missing")
	}
	return rotation.SelfRelease(ctx, p, platform, payload)
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
