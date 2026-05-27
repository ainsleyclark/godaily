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
	"net/http"
	"net/url"
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
// When req.MentionURN is a LinkedIn organisation URN and req.MentionDisplayName
// occurs (case-sensitive) in req.Text, the first occurrence is annotated as
// an inline mention so the rendered post links to that organisation page.
//
// The post URL is reconstructed from the x-restli-id response header
// (LinkedIn's REST convention for newly created resources).
//
// NOTE: the inline-mention shape (commentaryAnnotations with start/length/
// entity) reflects the LinkedIn Posts API v202601 contract as of writing.
// LinkedIn rolls API versions ~every quarter and renames fields between
// them; verify against the live docs before relying on mentions in
// production.
func (c *Client) Post(ctx context.Context, req platform.PostRequest) (platform.PostResponse, error) {
	body := postRequest{
		Author:                c.authorURN,
		Commentary:            req.Text,
		CommentaryAnnotations: buildAnnotations(req.Text, req.MentionURN, req.MentionDisplayName),
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

	endpoint := fmt.Sprintf(
		"%s/rest/organizationalEntityShareStatistics?q=organizationalEntity&organizationalEntity=%s&shares[0]=%s",
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
		return platform.Stats{}, fmt.Errorf("linkedin share statistics: %d %s: %s", resp.StatusCode, resp.Status, buf.String())
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

// buildAnnotations returns a single inline annotation pointing at urn for
// the first case-sensitive occurrence of displayName inside text. Returns
// nil when any input is empty or displayName is not found — in that case
// the post is sent without annotations and LinkedIn renders the text
// verbatim (same behaviour as before mention support existed).
func buildAnnotations(text, urn, displayName string) []inlineAnnotation {
	if text == "" || urn == "" || displayName == "" {
		return nil
	}
	idx := strings.Index(text, displayName)
	if idx < 0 {
		return nil
	}
	return []inlineAnnotation{{
		Start:  idx,
		Length: len(displayName),
		Entity: urn,
	}}
}

// feedURL builds the public URL for a published post. urn looks like
// "urn:li:share:7234567890123456789".
func feedURL(urn string) string {
	if urn == "" {
		return ""
	}
	return fmt.Sprintf("https://www.linkedin.com/feed/update/%s/", url.PathEscape(urn))
}
