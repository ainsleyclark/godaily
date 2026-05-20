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

// Package bluesky publishes posts to Bluesky using the AT Protocol XRPC
// HTTP API. Two requests are made per Post: createSession (auth) followed
// by repo.createRecord. The indigo SDK is intentionally avoided as it
// pulls in IPFS / libp2p dependencies that aren't justified here.
package bluesky

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/ainsleydev/webkit/pkg/util/httputil"
	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/gateway/social"
	"github.com/ainsleyclark/godaily/pkg/gohttp"
)

// defaultBaseURL is the public Bluesky PDS that user accounts live on by
// default. Self-hosted PDSes can override this via WithBaseURL.
const defaultBaseURL = "https://bsky.social"

// Client posts to Bluesky via the AT Protocol XRPC HTTP API.
type Client struct {
	handle      string
	appPassword string
	httpClient  *http.Client
	baseURL     string
	publicURL   string // base for converting at:// URIs to https:// post URLs
}

// New creates a new Bluesky Client. handle is the user's full handle (e.g.
// "godaily.bsky.social"); appPassword is an app-password generated via
// Bluesky settings (NOT the account password).
func New(handle, appPassword string) *Client {
	return &Client{
		handle:      handle,
		appPassword: appPassword,
		httpClient:  gohttp.New(gohttp.WithTimeout(15 * time.Second)),
		baseURL:     defaultBaseURL,
		publicURL:   "https://bsky.app",
	}
}

// Platform implements social.Poster.
func (c *Client) Platform() social.Platform {
	return social.PlatformBluesky
}

type (
	sessionResponse struct {
		AccessJWT string `json:"accessJwt"`
		DID       string `json:"did"`
	}
	createRecordResponse struct {
		URI string `json:"uri"`
		CID string `json:"cid"`
	}
)

// Post publishes text as a single post on the configured account. URLs in the
// text are annotated with AT Protocol link facets so they render as clickable
// hyperlinks in Bluesky clients.
func (c *Client) Post(ctx context.Context, text string) (social.Result, error) {
	session, err := c.createSession(ctx)
	if err != nil {
		return social.Result{}, errors.Wrap(err, "bluesky createSession")
	}

	record := map[string]any{
		"$type":     "app.bsky.feed.post",
		"text":      text,
		"createdAt": time.Now().UTC().Format(time.RFC3339),
		"langs":     []string{"en"},
	}
	if facets := buildFacets(text); len(facets) > 0 {
		record["facets"] = facets
	}

	body := map[string]any{
		"repo":       session.DID,
		"collection": "app.bsky.feed.post",
		"record":     record,
	}

	var out createRecordResponse
	if err := c.doJSON(ctx, "com.atproto.repo.createRecord", session.AccessJWT, body, &out); err != nil {
		return social.Result{}, errors.Wrap(err, "bluesky createRecord")
	}

	return social.Result{PostURL: c.postURLFromURI(out.URI)}, nil
}

func (c *Client) createSession(ctx context.Context) (sessionResponse, error) {
	body := map[string]string{
		"identifier": c.handle,
		"password":   c.appPassword,
	}
	var out sessionResponse
	if err := c.doJSON(ctx, "com.atproto.server.createSession", "", body, &out); err != nil {
		return sessionResponse{}, err
	}
	return out, nil
}

// doJSON POSTs body as JSON to /xrpc/<method>. When token is non-empty it is
// sent as a Bearer authorization header. Non-2xx responses become errors
// with the response body included for debugging.
func (c *Client) doJSON(ctx context.Context, method, token string, body, out any) error {
	buf, err := json.Marshal(body)
	if err != nil {
		return errors.Wrap(err, "marshalling request body")
	}

	url := c.baseURL + "/xrpc/" + method
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return errors.Wrap(err, "building request")
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "sending request")
	}
	defer resp.Body.Close()

	if !httputil.Is2xx(resp.StatusCode) {
		respBuf := new(bytes.Buffer)
		_, _ = respBuf.ReadFrom(resp.Body)
		return fmt.Errorf("%s: %d %s: %s", method, resp.StatusCode, resp.Status, respBuf.String())
	}

	if out == nil {
		return nil
	}

	if err = json.NewDecoder(resp.Body).Decode(out); err != nil {
		return errors.Wrap(err, "decoding response")
	}

	return nil
}

// urlRe matches http/https URLs in post text. Trailing punctuation is stripped
// separately so it isn't captured as part of the link target.
var urlRe = regexp.MustCompile(`https?://[^\s]+`)

type facetIndex struct {
	ByteStart int `json:"byteStart"`
	ByteEnd   int `json:"byteEnd"`
}

type facetFeature struct {
	Type string `json:"$type"`
	URI  string `json:"uri"`
}

type facet struct {
	Index    facetIndex     `json:"index"`
	Features []facetFeature `json:"features"`
}

// buildFacets returns AT Protocol link facets for every URL found in text.
// Byte offsets are computed over the UTF-8 encoding of text, as required by
// the AT Protocol richtext spec.
func buildFacets(text string) []facet {
	b := []byte(text)
	matches := urlRe.FindAllIndex(b, -1)
	if len(matches) == 0 {
		return nil
	}
	out := make([]facet, 0, len(matches))
	for _, m := range matches {
		uri := strings.TrimRight(string(b[m[0]:m[1]]), ".,;:!?)'\"")
		out = append(out, facet{
			Index: facetIndex{ByteStart: m[0], ByteEnd: m[0] + len(uri)},
			Features: []facetFeature{{
				Type: "app.bsky.richtext.facet#link",
				URI:  uri,
			}},
		})
	}
	return out
}

// postURLFromURI converts an at:// URI ("at://did:plc:xxx/app.bsky.feed.post/<rkey>")
// to the public web URL on bsky.app. Returns "" if the URI is unparseable.
func (c *Client) postURLFromURI(uri string) string {
	const prefix = "at://"
	if !strings.HasPrefix(uri, prefix) {
		return ""
	}

	parts := strings.SplitN(strings.TrimPrefix(uri, prefix), "/", 3)
	if len(parts) != 3 {
		return ""
	}

	rKey := parts[2]
	return fmt.Sprintf("%s/profile/%s/post/%s", c.publicURL, c.handle, rKey)
}
