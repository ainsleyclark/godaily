// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/source/ingest"
)

// GitHub defines the type that implements news.Fetcher for golang/go proposals.
type GitHub struct {
	endpoints []ghEndpoint
	token     string
}

var _ news.Fetcher = &GitHub{}

func init() {
	news.Register(news.SourceGitHub, func(cfg env.Config) news.Fetcher { return NewGitHub(cfg) })
}

const ghBase = "https://api.github.com/repos/golang/go/issues"

// ghDefaultEndpoints defines the four proposal lifecycle stages to fetch, ordered
// so the most specific tag wins when the same issue appears across multiple results.
//
// Accepted proposals carry both "Proposal" and "Proposal-Accepted" labels, so
// fetching accepted first ensures they get TagProposalAccepted rather than TagProposal
// when the open-proposals call is processed.
var ghDefaultEndpoints = []ghEndpoint{
	{url: ghBase + "?labels=Proposal-Accepted&state=open&sort=updated&per_page=20", tag: news.TagProposalAccepted},
	{url: ghBase + "?labels=Proposal-FinalCommentPeriod&state=open&sort=updated&per_page=10", tag: news.TagProposal},
	{url: ghBase + "?labels=Proposal&state=open&sort=updated&per_page=30", tag: news.TagProposal},
	{url: ghBase + "?labels=Proposal-Accepted&state=closed&sort=updated&per_page=10", tag: news.TagProposalShipped},
}

type ghEndpoint struct {
	url string
	tag news.Tag
}

// NewGitHub creates a GitHub client. It uses cfg.GitHubToken as a Bearer token
// if present (raises rate limit from 60 to 5000/hr).
func NewGitHub(cfg env.Config) *GitHub {
	return &GitHub{
		endpoints: ghDefaultEndpoints,
		token:     cfg.GitHubToken,
	}
}

// Fetch retrieves Go proposal issues from the golang/go repository across four
// lifecycle stages and returns them as a deduplicated list of news items.
func (g GitHub) Fetch(ctx context.Context) ([]news.Item, error) {
	var (
		collected []ghIssue
		seen      = make(map[string]bool)
	)

	var hdrs []http.Header
	if g.token != "" {
		hdrs = []http.Header{{"Authorization": {"Bearer " + g.token}}}
	}

	for _, ep := range g.endpoints {
		issues, err := ingest.Fetch[[]ghIssue](ctx, ep.url, "github", json.Unmarshal, hdrs...)
		if err != nil {
			return nil, err
		}
		for i := range issues {
			if seen[issues[i].HTMLURL] {
				continue
			}
			seen[issues[i].HTMLURL] = true
			issues[i].tag = ep.tag
			collected = append(collected, issues[i])
		}
	}

	return ingest.TransformAll(ctx, collected), nil
}

func (i ghIssue) ShouldInclude() bool   { return true }
func (i ghIssue) EnrichmentURL() string { return i.HTMLURL }

// Transform maps a ghIssue to a news.Item using the tag stored on the issue
// (set in Fetch from the originating endpoint).
func (i ghIssue) Transform() news.Item {
	return news.Item{
		Source: news.SourceGitHub,
		Title:  i.Title,
		URL:    i.HTMLURL,
		Author: &news.Author{
			Username:   i.User.Login,
			AvatarURL:  i.User.AvatarURL,
			ProfileURL: i.User.HTMLURL,
		},
		Snippet:   ghSnippet(i.Body, i.Milestone),
		Tag:       i.tag,
		Comments:  i.Comments,
		Score:     news.ScoreOf(news.SourceGitHub, i.tag, float64(i.Reactions.PlusOne), true),
		Published: i.publishedAt(),
	}
}

// publishedAt returns the date the issue became newsworthy. For accepted
// proposals that is the acceptance, not the original filing: these issues are
// created years before they are accepted, so created_at would always fall
// outside the digest's collection window and the item would never surface.
// updated_at is the closest available proxy for the acceptance (the
// Proposal-Accepted endpoints sort by it), and the items upsert freezes
// published on first insert, so an accepted proposal lands in exactly one
// digest regardless of later activity. Open proposals keep created_at — a
// freshly filed proposal is caught at creation.
func (i ghIssue) publishedAt() time.Time {
	if i.tag == news.TagProposalAccepted && !i.UpdatedAt.IsZero() {
		return i.UpdatedAt
	}
	return i.CreatedAt
}

var (
	// goVersionRe matches milestone titles like "Go1.25" or "Go1.25.1".
	goVersionRe = regexp.MustCompile(`^Go\d+\.\d+`)
	// mdNoiseRe strips common markdown syntax characters from issue bodies.
	mdNoiseRe = regexp.MustCompile("[#*`]+")
)

// ghSnippet builds a snippet from the issue body. If the issue has a versioned
// milestone (e.g. "Go1.27"), the snippet is prefixed with "Targeting Go 1.27 — ".
// Milestones like "Backlog" produce no prefix.
func ghSnippet(body string, m *ghMilestone) string {
	clean := strings.TrimSpace(mdNoiseRe.ReplaceAllString(body, " "))
	clean = strings.Join(strings.Fields(clean), " ")
	if len(clean) > 150 {
		clean = clean[:150]
	}
	if m != nil && goVersionRe.MatchString(m.Title) {
		return "Targeting Go " + strings.TrimPrefix(m.Title, "Go") + " \u2014 " + clean
	}
	return clean
}

type (
	ghIssue struct {
		Title     string       `json:"title"`
		HTMLURL   string       `json:"html_url"`
		Body      string       `json:"body"`
		User      ghUser       `json:"user"`
		Milestone *ghMilestone `json:"milestone"`
		Comments  int          `json:"comments"`
		Reactions ghReactions  `json:"reactions"`
		CreatedAt time.Time    `json:"created_at"`
		UpdatedAt time.Time    `json:"updated_at"`
		tag       news.Tag     // populated by Fetch from the endpoint that returned this issue
	}
	ghUser struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
		HTMLURL   string `json:"html_url"`
	}
	ghMilestone struct {
		Title string `json:"title"`
	}
	ghReactions struct {
		PlusOne int `json:"+1"`
	}
)
