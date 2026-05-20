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
	"time"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/gateway/social"
)

const (
	defaultBaseURL = "https://api.linkedin.com"
	// DefaultAPIVersion is the LinkedIn-Version header value used when none
	// is provided to New. LinkedIn retires versions ~12 months after release,
	// so callers should override this via LINKEDIN_API_VERSION in production
	// rather than relying on the bundled default rolling forward.
	DefaultAPIVersion = "202601"
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
// DefaultAPIVersion.
func New(token, authorURN, apiVersion string) *Client {
	if apiVersion == "" {
		apiVersion = DefaultAPIVersion
	}
	return &Client{
		token:      token,
		authorURN:  authorURN,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		baseURL:    defaultBaseURL,
		apiVersion: apiVersion,
	}
}

// Platform implements social.Poster.
func (c *Client) Platform() social.Platform {
	return social.PlatformLinkedIn
}

// postRequest is the body sent to /rest/posts. Field names follow the
// platform's documented shape.
type postRequest struct {
	Author                    string       `json:"author"`
	Commentary                string       `json:"commentary"`
	Visibility                string       `json:"visibility"`
	Distribution              distribution `json:"distribution"`
	LifecycleState            string       `json:"lifecycleState"`
	IsReshareDisabledByAuthor bool         `json:"isReshareDisabledByAuthor"`
}

type distribution struct {
	FeedDistribution               string   `json:"feedDistribution"`
	TargetEntities                 []string `json:"targetEntities"`
	ThirdPartyDistributionChannels []string `json:"thirdPartyDistributionChannels"`
}

// Post publishes text to the configured organisation's feed.
//
// The post URL is reconstructed from the x-restli-id response header
// (LinkedIn's REST convention for newly created resources).
func (c *Client) Post(ctx context.Context, text string) (social.Result, error) {
	body := postRequest{
		Author:         c.authorURN,
		Commentary:     text,
		Visibility:     "PUBLIC",
		LifecycleState: "PUBLISHED",
		Distribution: distribution{
			FeedDistribution:               "MAIN_FEED",
			TargetEntities:                 []string{},
			ThirdPartyDistributionChannels: []string{},
		},
	}

	buf, err := json.Marshal(body)
	if err != nil {
		return social.Result{}, errors.Wrap(err, "marshalling post body")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/rest/posts", bytes.NewReader(buf))
	if err != nil {
		return social.Result{}, errors.Wrap(err, "building request")
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("LinkedIn-Version", c.apiVersion)
	req.Header.Set("X-Restli-Protocol-Version", "2.0.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return social.Result{}, errors.Wrap(err, "sending request")
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBuf := new(bytes.Buffer)
		_, _ = respBuf.ReadFrom(resp.Body)
		return social.Result{}, fmt.Errorf("linkedin /rest/posts: %d %s: %s", resp.StatusCode, resp.Status, respBuf.String())
	}

	urn := resp.Header.Get("x-restli-id")
	return social.Result{PostURL: feedURL(urn)}, nil
}

// feedURL builds the public URL for a published post. urn looks like
// "urn:li:share:7234567890123456789".
func feedURL(urn string) string {
	if urn == "" {
		return ""
	}
	return fmt.Sprintf("https://www.linkedin.com/feed/update/%s/", url.PathEscape(urn))
}
