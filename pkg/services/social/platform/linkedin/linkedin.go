// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package linkedin publishes posts to a LinkedIn organisation page via the
// /rest/posts endpoint. The token must carry the w_organization_social
// scope and be associated with an admin of the target organisation.
package linkedin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/ainsleydev/webkit/pkg/util/httputil"
	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/platform"
	"github.com/ainsleyclark/godaily/pkg/util/gohttp"
)

const (
	defaultBaseURL = "https://api.linkedin.com"
	// defaultAPIVersion is the LinkedIn-Version header value used when none
	// is provided to New. LinkedIn retires versions ~12 months after release,
	// so callers should override this via LINKEDIN_API_VERSION in production
	// rather than relying on the bundled default rolling forward.
	defaultAPIVersion = "202601"
)

// Client posts to LinkedIn via the /rest/posts endpoint.
type Client struct {
	token      string
	authorURN  string
	httpClient *http.Client
	baseURL    string
	apiVersion string
}

// New creates a new LinkedIn Client. authorURN is the URN of the entity that
// authored the post (e.g. "urn:li:organization:12345" for an organisation
// page). apiVersion sets the LinkedIn-Version header — pass "" to use
// defaultAPIVersion.
func New(token, authorURN string) *Client {
	return &Client{
		token:      token,
		authorURN:  authorURN,
		httpClient: gohttp.New(gohttp.WithTimeout(15 * time.Second)),
		baseURL:    defaultBaseURL,
		apiVersion: defaultAPIVersion,
	}
}

// Platform implements platform.Poster.
func (c *Client) Platform() social.Platform {
	return social.LinkedIn
}

type (
	// postRequest is the body sent to /rest/posts. Field names follow the
	// platform's documented shape.
	postRequest struct {
		Author                    string             `json:"author"`
		Commentary                string             `json:"commentary"`
		CommentaryAnnotations     []inlineAnnotation `json:"commentaryAnnotations,omitempty"`
		Visibility                string             `json:"visibility"`
		Distribution              distribution       `json:"distribution"`
		LifecycleState            string             `json:"lifecycleState"`
		IsReshareDisabledByAuthor bool               `json:"isReshareDisabledByAuthor"`
	}
	distribution struct {
		FeedDistribution               string   `json:"feedDistribution"`
		TargetEntities                 []string `json:"targetEntities"`
		ThirdPartyDistributionChannels []string `json:"thirdPartyDistributionChannels"`
	}
	// inlineAnnotation marks a (start, length) range of commentary as a
	// link to the entity in entity. start is a zero-based UTF-16 code-unit
	// offset and length is in code units — matching LinkedIn's convention
	// for the Posts API.
	inlineAnnotation struct {
		Start  int    `json:"start"`
		Length int    `json:"length"`
		Entity string `json:"entity"`
	}
)

// Post publishes the request text to the configured organisation's feed.
// Mentions whose Platform is LinkedIn and whose DisplayName occurs
// (case-sensitive) in req.Text are attached as inline annotations so the
// rendered post links to the referenced entity. Mentions whose name
// can't be matched in the text are logged at WARN and dropped — the
// post still goes through, just without a tag for that entity.
//
// The post URL is reconstructed from the x-restli-id response header
// (LinkedIn's REST convention for newly created resources).
//
// NOTE: the inline-mention shape (commentaryAnnotations with start /
// length / entity) reflects the LinkedIn Posts API v202601 contract as
// of writing. LinkedIn rolls API versions ~every quarter and renames
// fields between them; verify against the live docs before relying on
// mentions in production.
func (c *Client) Post(ctx context.Context, req platform.PostRequest) (platform.PostResponse, error) {
	annotations, missed := buildAnnotations(req.Text, req.Mentions)
	for _, m := range missed {
		slog.WarnContext(
			ctx, "LinkedIn mention dropped: display name not found in post text",
			"display_name", m.DisplayName, "handle", m.Handle,
		)
	}

	body := postRequest{
		Author:                c.authorURN,
		Commentary:            req.Text,
		CommentaryAnnotations: annotations,
		Visibility:            "PUBLIC",
		LifecycleState:        "PUBLISHED",
		Distribution: distribution{
			FeedDistribution:               "MAIN_FEED",
			TargetEntities:                 []string{},
			ThirdPartyDistributionChannels: []string{},
		},
	}

	buf, err := json.Marshal(body)
	if err != nil {
		return platform.PostResponse{}, errors.Wrap(err, "marshalling post body")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/rest/posts", bytes.NewReader(buf))
	if err != nil {
		return platform.PostResponse{}, errors.Wrap(err, "building request")
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.token)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("LinkedIn-Version", c.apiVersion)
	httpReq.Header.Set("X-Restli-Protocol-Version", "2.0.0")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return platform.PostResponse{}, errors.Wrap(err, "sending request")
	}
	defer resp.Body.Close()

	if !httputil.Is2xx(resp.StatusCode) {
		respBuf := new(bytes.Buffer)
		_, _ = respBuf.ReadFrom(resp.Body)
		return platform.PostResponse{}, fmt.Errorf("linkedin /rest/posts: %d %s: %s", resp.StatusCode, resp.Status, respBuf.String())
	}

	urn := resp.Header.Get("x-restli-id")
	return platform.PostResponse{PostURL: feedURL(urn)}, nil
}

// Stats fetches engagement counts for a LinkedIn organisation post.
// postURL must be the canonical feed URL stored in social_posts.post_url,
// e.g. https://www.linkedin.com/feed/update/urn:li:share:7234567890/.
// Requires the token to carry r_organization_social scope.
func (c *Client) Stats(ctx context.Context, postURL string) (platform.Stats, error) {
	shareURN, err := urnFromPostURL(postURL)
	if err != nil {
		return platform.Stats{}, errors.Wrap(err, "extracting share URN from post URL")
	}

	// The request declares X-Restli-Protocol-Version 2.0.0, so array query
	// params must use the reduced List(...) encoding. The legacy indexed
	// form (shares[0]=...) is rejected under 2.0.0 with a 400
	// QUERY_PARAM_NOT_ALLOWED on fieldPath "shares[0]".
	endpoint := fmt.Sprintf(
		"%s/rest/organizationalEntityShareStatistics?q=organizationalEntity&organizationalEntity=%s&shares=List(%s)",
		c.baseURL,
		url.QueryEscape(c.authorURN),
		url.QueryEscape(shareURN),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return platform.Stats{}, errors.Wrap(err, "building request")
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("LinkedIn-Version", c.apiVersion)
	req.Header.Set("X-Restli-Protocol-Version", "2.0.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return platform.Stats{}, errors.Wrap(err, "sending request")
	}
	defer resp.Body.Close()

	if !httputil.Is2xx(resp.StatusCode) {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)
		body := buf.String()
		// A deleted post leaves its share URN in our store but LinkedIn can
		// no longer resolve a backing activity for it, returning a 400 with
		// this message. Surface it as a terminal "gone" condition so the
		// caller can skip it quietly instead of treating it as a failure.
		if resp.StatusCode == http.StatusBadRequest &&
			strings.Contains(body, "Unable to get activityIds from any of the given shares") {
			return platform.Stats{}, fmt.Errorf("linkedin share statistics: %s: %w", body, platform.ErrPostUnavailable)
		}
		return platform.Stats{}, fmt.Errorf("linkedin share statistics: %d %s: %s", resp.StatusCode, resp.Status, body)
	}

	var out struct {
		Elements []struct {
			TotalShareStatistics struct {
				LikeCount       int64 `json:"likeCount"`
				CommentCount    int64 `json:"commentCount"`
				ShareCount      int64 `json:"shareCount"`
				ImpressionCount int64 `json:"impressionCount"`
				ClickCount      int64 `json:"clickCount"`
			} `json:"totalShareStatistics"`
		} `json:"elements"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return platform.Stats{}, errors.Wrap(err, "decoding response")
	}
	if len(out.Elements) == 0 {
		return platform.Stats{}, nil
	}
	s := out.Elements[0].TotalShareStatistics
	return platform.Stats{
		Likes:       s.LikeCount,
		Reposts:     s.ShareCount,
		Comments:    s.CommentCount,
		Impressions: s.ImpressionCount,
	}, nil
}

// urnFromPostURL extracts the share URN from a LinkedIn feed update URL.
// URL form: https://www.linkedin.com/feed/update/urn:li:share:7234567890/
func urnFromPostURL(postURL string) (string, error) {
	u, err := url.Parse(postURL)
	if err != nil {
		return "", errors.Wrap(err, "parsing post URL")
	}
	// Path: /feed/update/<urn>/
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 3 {
		return "", fmt.Errorf("unexpected LinkedIn feed URL format: %s", postURL)
	}
	urn, err := url.PathUnescape(parts[len(parts)-1])
	if err != nil {
		return "", errors.Wrap(err, "unescaping URN")
	}
	if !strings.HasPrefix(urn, "urn:li:") {
		return "", fmt.Errorf("extracted value is not a LinkedIn URN: %q", urn)
	}
	return urn, nil
}

// buildAnnotations resolves mentions against text into a list of inline
// annotations and a list of mentions that couldn't be inlined (because
// their DisplayName didn't appear in text, case-sensitively). Only
// LinkedIn-platform mentions are considered; everything else is silently
// skipped.
//
// Conflict resolution: when two mentions could match overlapping ranges
// (e.g. one annotates "Go" and another annotates "Go Blog"), the longer
// match wins. On equal length the earlier mention wins. Each URN is
// annotated at most once per post — additional occurrences of the same
// name are ignored.
func buildAnnotations(text string, mentions []social.Mention) ([]inlineAnnotation, []social.Mention) {
	if text == "" || len(mentions) == 0 {
		return nil, nil
	}

	// Collect the first case-sensitive match for each LinkedIn mention.
	type candidate struct {
		start  int
		length int
		entity string
	}
	var (
		candidates []candidate
		missed     []social.Mention
	)
	for _, m := range mentions {
		if m.Platform != social.LinkedIn || m.Handle == "" || m.DisplayName == "" {
			continue
		}
		idx := strings.Index(text, m.DisplayName)
		if idx < 0 {
			missed = append(missed, m)
			continue
		}
		candidates = append(candidates, candidate{
			start:  idx,
			length: len(m.DisplayName),
			entity: m.Handle,
		})
	}
	if len(candidates) == 0 {
		return nil, missed
	}

	// Resolve overlaps: longest match wins, ties broken by earlier
	// position.
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].length != candidates[j].length {
			return candidates[i].length > candidates[j].length
		}
		return candidates[i].start < candidates[j].start
	})
	accepted := make([]candidate, 0, len(candidates))
	for _, cand := range candidates {
		overlaps := false
		for _, a := range accepted {
			if cand.start < a.start+a.length && a.start < cand.start+cand.length {
				overlaps = true
				break
			}
		}
		if !overlaps {
			accepted = append(accepted, cand)
		}
	}

	// LinkedIn expects annotations in document order.
	sort.SliceStable(accepted, func(i, j int) bool {
		return accepted[i].start < accepted[j].start
	})
	out := make([]inlineAnnotation, len(accepted))
	for i, a := range accepted {
		out[i] = inlineAnnotation{Start: a.start, Length: a.length, Entity: a.entity}
	}
	return out, missed
}

// feedURL builds the public URL for a published post. urn looks like
// "urn:li:share:7234567890123456789".
func feedURL(urn string) string {
	if urn == "" {
		return ""
	}
	return fmt.Sprintf("https://www.linkedin.com/feed/update/%s/", url.PathEscape(urn))
}
