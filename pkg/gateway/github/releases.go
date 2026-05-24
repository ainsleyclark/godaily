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

// Package github is the HTTP gateway for the GitHub REST API. It is
// intentionally separate from pkg/source/github.go (which fetches Go
// proposal issues) — that file is a news Fetcher, this package exposes
// primitives the rest of the app uses (releases, commits, ...).
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/gohttp"
	"github.com/ainsleyclark/godaily/pkg/services/social/candidates"
)

// apiBase is the GitHub REST API root.
const apiBase = "https://api.github.com"

// Releases fetches GitHub releases for a single repository.
type Releases struct {
	owner  string
	repo   string
	token  string
	client *http.Client
}

// NewReleases returns a Releases client. An empty token works against the
// public API at a lower rate limit; a personal access token raises the
// limit to 5000/hr.
func NewReleases(owner, repo, token string) *Releases {
	return &Releases{
		owner:  owner,
		repo:   repo,
		token:  token,
		client: gohttp.New(),
	}
}

// Ensure the type satisfies the candidates contract.
var _ candidates.ReleaseFetcher = (*Releases)(nil)

// LatestReleases returns up to 10 of the most recent releases, newest
// first. The candidate filters drafts and pre-releases itself.
func (r *Releases) LatestReleases(ctx context.Context) ([]candidates.GitHubRelease, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases?per_page=10", apiBase, r.owner, r.repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "github: new request")
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if r.token != "" {
		req.Header.Set("Authorization", "Bearer "+r.token)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "github: do request")
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, errors.Errorf("github: %s -> %d %s", url, resp.StatusCode, string(body))
	}

	var raw []ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, errors.Wrap(err, "github: decode releases")
	}

	out := make([]candidates.GitHubRelease, 0, len(raw))
	for _, r := range raw {
		out = append(out, candidates.GitHubRelease{
			TagName:     r.TagName,
			Name:        r.Name,
			HTMLURL:     r.HTMLURL,
			Body:        r.Body,
			PublishedAt: r.PublishedAt,
			Prerelease:  r.Prerelease,
			Draft:       r.Draft,
		})
	}
	return out, nil
}

type ghRelease struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	HTMLURL     string    `json:"html_url"`
	Body        string    `json:"body"`
	PublishedAt time.Time `json:"published_at"`
	Prerelease  bool      `json:"prerelease"`
	Draft       bool      `json:"draft"`
}
