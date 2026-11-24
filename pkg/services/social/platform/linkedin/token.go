// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package linkedin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ainsleydev/webkit/pkg/util/httputil"
	"github.com/pkg/errors"
)

const (
	// defaultOAuthURL is LinkedIn's OAuth 2.0 token endpoint, used to
	// exchange a refresh token for a fresh access token.
	defaultOAuthURL = "https://www.linkedin.com/oauth/v2/accessToken"

	// refreshTokenWarnWindow is how far ahead of the refresh token's own
	// expiry we start logging warnings. LinkedIn refresh tokens have a fixed
	// lifetime (365 days from the original consent) that does NOT roll
	// forward when used, so someone has to re-authorise roughly once a year.
	refreshTokenWarnWindow = 30 * 24 * time.Hour
)

// Option configures a Client.
type Option func(*Client)

// WithRefresh enables automatic access-token refresh. When LinkedIn rejects
// a request with a 401 (typically EXPIRED_ACCESS_TOKEN, as access tokens
// only live 60 days), the client exchanges refreshToken for a new access
// token and retries the request once. If no initial access token was
// supplied to New, one is fetched before the first request.
//
// The option is a no-op unless all three values are non-empty, so it is
// safe to pass straight from optional config.
//
// The refreshed token is held in memory only; each new process (e.g. each
// serverless cold start) refreshes again on its first 401. Refresh tokens
// are not rotated by LinkedIn, so there is nothing to persist.
func WithRefresh(refreshToken, clientID, clientSecret string) Option {
	return func(c *Client) {
		if refreshToken == "" || clientID == "" || clientSecret == "" {
			return
		}
		c.refresh = &refresher{
			refreshToken: refreshToken,
			clientID:     clientID,
			clientSecret: clientSecret,
			oauthURL:     defaultOAuthURL,
		}
	}
}

// refresher holds the credentials needed to mint new access tokens.
type refresher struct {
	refreshToken string
	clientID     string
	clientSecret string
	oauthURL     string
}

// tokenResponse is the body returned by the OAuth token endpoint.
type tokenResponse struct {
	AccessToken           string `json:"access_token"`
	ExpiresIn             int64  `json:"expires_in"`
	RefreshToken          string `json:"refresh_token"`
	RefreshTokenExpiresIn int64  `json:"refresh_token_expires_in"`
}

// tokenState guards the current access token so concurrent requests share
// a single refresh.
type tokenState struct {
	mu    sync.Mutex
	token string
}

func (s *tokenState) get() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.token
}

// currentToken returns the access token to use, fetching one first when the
// client was built with refresh credentials but no initial access token.
func (c *Client) currentToken(ctx context.Context) (string, error) {
	if tok := c.tokens.get(); tok != "" || c.refresh == nil {
		return tok, nil
	}
	return c.refreshToken(ctx, "")
}

// refreshToken exchanges the refresh token for a new access token. stale is
// the token the caller saw rejected: if another goroutine has already
// replaced it, the newer token is returned without a second round-trip.
func (c *Client) refreshToken(ctx context.Context, stale string) (string, error) {
	c.tokens.mu.Lock()
	defer c.tokens.mu.Unlock()

	if c.tokens.token != "" && c.tokens.token != stale {
		return c.tokens.token, nil
	}

	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {c.refresh.refreshToken},
		"client_id":     {c.refresh.clientID},
		"client_secret": {c.refresh.clientSecret},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.refresh.oauthURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", errors.Wrap(err, "building token refresh request")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "sending token refresh request")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "reading token refresh response")
	}
	if !httputil.Is2xx(resp.StatusCode) {
		// Never include the request (it carries the client secret); the
		// response body only holds LinkedIn's error description.
		return "", fmt.Errorf("linkedin token refresh: %d %s: %s", resp.StatusCode, resp.Status, string(body))
	}

	var out tokenResponse
	if err = json.Unmarshal(body, &out); err != nil {
		return "", errors.Wrap(err, "decoding token refresh response")
	}
	if out.AccessToken == "" {
		return "", errors.New("linkedin token refresh: response did not contain an access_token")
	}

	c.tokens.token = out.AccessToken
	if out.RefreshToken != "" {
		c.refresh.refreshToken = out.RefreshToken
	}

	attrs := []any{"access_token_expires_in", time.Duration(out.ExpiresIn) * time.Second}
	if out.RefreshTokenExpiresIn > 0 {
		left := time.Duration(out.RefreshTokenExpiresIn) * time.Second
		attrs = append(attrs, "refresh_token_expires_in", left)
		if left < refreshTokenWarnWindow {
			slog.WarnContext(
				ctx, "LinkedIn refresh token expires soon: re-authorise the app and update LINKEDIN_REFRESH_TOKEN",
				attrs...,
			)
			return out.AccessToken, nil
		}
	}
	slog.InfoContext(ctx, "LinkedIn access token refreshed", attrs...)

	return out.AccessToken, nil
}

// do sends the request produced by build, authorising it with the current
// access token. On a 401, when refresh is configured, it refreshes the
// token and retries exactly once. build is called once per attempt so
// request bodies can be re-read.
func (c *Client) do(ctx context.Context, build func() (*http.Request, error)) (*http.Response, error) {
	tok, err := c.currentToken(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "obtaining access token")
	}

	resp, err := c.send(build, tok)
	if err != nil || resp.StatusCode != http.StatusUnauthorized || c.refresh == nil {
		return resp, err
	}

	// Drain and close the rejected response before retrying.
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()

	slog.InfoContext(ctx, "LinkedIn rejected access token, refreshing")
	tok, err = c.refreshToken(ctx, tok)
	if err != nil {
		return nil, errors.Wrap(err, "refreshing access token after 401")
	}
	return c.send(build, tok)
}

func (c *Client) send(build func() (*http.Request, error), token string) (*http.Response, error) {
	req, err := build()
	if err != nil {
		return nil, errors.Wrap(err, "building request")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("LinkedIn-Version", c.apiVersion)
	req.Header.Set("X-Restli-Protocol-Version", "2.0.0")
	return c.httpClient.Do(req)
}
