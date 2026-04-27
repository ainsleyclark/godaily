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

package source

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/ainsleyclark/godaily/internal/news"
)

// GitHub defines the type that implements news.Fetcher for golang/go proposals.
type GitHub struct {
	endpoints []ghEndpoint
	token     string
}

var _ news.Fetcher = &GitHub{}

func init() {
	news.Register(news.SourceGitHub, NewGitHub())
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

// NewGitHub creates a GitHub client. It reads GITHUB_TOKEN from the environment
// and uses it as a Bearer token if present (raises rate limit from 60 to 5000/hr).
func NewGitHub() *GitHub {
	return &GitHub{
		endpoints: ghDefaultEndpoints,
		token:     os.Getenv("GITHUB_TOKEN"),
	}
}

// Fetch retrieves Go proposal issues from the golang/go repository across four
// lifecycle stages and returns them as a deduplicated list of news items.
func (g GitHub) Fetch(ctx context.Context) ([]news.Item, error) {
	var (
		items []news.Item
		seen  = make(map[string]bool)
	)

	var hdrs []http.Header
	if g.token != "" {
		hdrs = []http.Header{{"Authorization": {"Bearer " + g.token}}}
	}

	for _, ep := range g.endpoints {
		issues, err := fetch[[]ghIssue](ctx, ep.url, "github", json.Unmarshal, hdrs...)
		if err != nil {
			return nil, err
		}
		for _, issue := range issues {
			if seen[issue.HTMLURL] {
				continue
			}
			seen[issue.HTMLURL] = true
			items = append(items, issue.transform(ep.tag))
		}
	}

	return items, nil
}

// transform maps a ghIssue to a news.Item using the given tag.
func (i ghIssue) transform(tag news.Tag) news.Item {
	return news.Item{
		Source:    news.SourceGitHub,
		Title:     i.Title,
		URL:       i.HTMLURL,
		Author:    i.User.Login,
		Snippet:   ghSnippet(i.Body, i.Milestone),
		Tag:       tag,
		Comments:  i.Comments,
		Score:     news.ScoreOf(news.SourceGitHub, tag, float64(i.Reactions.PlusOne), true),
		Published: i.CreatedAt,
	}
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
	}
	ghUser struct {
		Login string `json:"login"`
	}
	ghMilestone struct {
		Title string `json:"title"`
	}
	ghReactions struct {
		PlusOne int `json:"+1"`
	}
)
