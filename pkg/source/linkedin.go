// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"context"
	"encoding/json"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/source/ingest"
	"github.com/pkg/errors"
)

// LinkedIn fetches recent #golang posts from the LinkedIn Voyager/Dash GraphQL
// API using a browser session cookie. The Voyager API is LinkedIn's internal web
// API; it is not officially documented and requires a real user session (see
// LINKEDIN_COOKIE in .env). The CSRF token is auto-extracted from the JSESSIONID
// cookie value.
type LinkedIn struct {
	url    string
	cookie string // full Cookie header value copied from browser DevTools
	client *http.Client
}

var _ news.Fetcher = &LinkedIn{}

func init() {
	news.Register(news.SourceLinkedIn, func(cfg env.Config) news.Fetcher { return NewLinkedIn(cfg) })
}

const (
	// graphql endpoint confirmed working via browser DevTools (2026-06-03).
	linkedInGraphQLURL = "https://www.linkedin.com/voyager/api/graphql" +
		"?includeWebMetadata=true" +
		"&variables=(start:0,origin:OTHER,query:(keywords:%23golang,flagshipSearchIntent:SEARCH_SRP,queryParameters:List((key:resultType,value:List(CONTENT))),includeFiltersInResponse:false),count:20)" +
		"&queryId=voyagerSearchDashClusters.843215f2a3455f1bed85762a45d71be8"

	linkedInMinLikes    = 5
	linkedInTitleMaxLen = 80

	// Entity types returned in the normalized JSON included array.
	linkedInTypeUpdate         = "com.linkedin.voyager.dash.feed.Update"
	linkedInTypeSocialCounts   = "com.linkedin.voyager.dash.feed.SocialActivityCounts"
	linkedInSocialCountsPrefix = "urn:li:fsd_socialActivityCounts:"
)

// linkedInNoRedirectClient never follows redirects so a LinkedIn auth failure
// surfaces as a clean 302 error rather than an uninformative redirect loop.
var linkedInNoRedirectClient = &http.Client{
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	},
	Timeout: 30 * time.Second,
}

// NewLinkedIn creates a LinkedIn hashtag-feed client.
func NewLinkedIn(cfg env.Config) *LinkedIn {
	return &LinkedIn{
		url:    linkedInGraphQLURL,
		cookie: cfg.LinkedInCookie,
		client: linkedInNoRedirectClient,
	}
}

// Fetch retrieves recent #golang posts from the LinkedIn Voyager/Dash search API.
func (l *LinkedIn) Fetch(ctx context.Context) ([]news.Item, error) {
	if l.cookie == "" {
		return nil, errors.New("linkedin: LINKEDIN_COOKIE is not set — copy the full Cookie header from a LinkedIn network request in browser DevTools")
	}

	csrf := linkedInCSRFFromCookie(l.cookie)
	if csrf == "" {
		return nil, errors.New("linkedin: JSESSIONID not found in LINKEDIN_COOKIE — ensure you copied the full Cookie header including JSESSIONID")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "linkedin request creation failed")
	}
	req.Header.Set("Cookie", l.cookie)
	req.Header.Set("csrf-token", csrf)
	req.Header.Set("Accept", "application/vnd.linkedin.normalized+json+2.1")
	req.Header.Set("Accept-Language", "en-GB,en-US;q=0.9,en;q=0.8")
	req.Header.Set("X-Restli-Protocol-Version", "2.0.0")
	req.Header.Set("X-Li-Lang", "en_US")
	req.Header.Set("X-Li-Page-Instance", "urn:li:page:d_flagship3_search_srp_content;00000000-0000-0000-0000-000000000001")
	req.Header.Set("X-Li-Pem-Metadata", "Voyager - Content SRP=search-results")
	req.Header.Set("x-li-track", `{"clientVersion":"1.13.44541","mpVersion":"1.13.44541","osName":"web","timezoneOffset":0,"timezone":"UTC","deviceFormFactor":"DESKTOP","mpName":"voyager-web","displayDensity":2,"displayWidth":1920,"displayHeight":1080}`)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://www.linkedin.com/search/results/content/?keywords=%23golang")

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "fetch linkedin")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusMovedPermanently {
		return nil, errors.Errorf("linkedin: authentication failed (got %d) — refresh LINKEDIN_COOKIE in .env", resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.Errorf("unexpected status code from linkedin: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "reading linkedin response")
	}

	posts, err := linkedInParseResponse(data)
	if err != nil {
		return nil, err
	}

	return ingest.TransformAll(ctx, posts), nil
}

// linkedInParseResponse unmarshals the normalized JSON response, separates
// Update and SocialActivityCounts entities from the included array, and joins
// them by the activity URN to produce self-contained linkedInPost values.
func linkedInParseResponse(data []byte) ([]linkedInPost, error) {
	var raw struct {
		Included []json.RawMessage `json:"included"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, errors.Wrap(err, "parsing linkedin response")
	}

	var updates []linkedInDashUpdate
	countsByURN := make(map[string]linkedInDashSocialCounts)

	for _, msg := range raw.Included {
		var t struct {
			Type string `json:"$type"`
		}
		if err := json.Unmarshal(msg, &t); err != nil {
			continue
		}
		switch t.Type {
		case linkedInTypeUpdate:
			var u linkedInDashUpdate
			if err := json.Unmarshal(msg, &u); err == nil {
				updates = append(updates, u)
			}
		case linkedInTypeSocialCounts:
			var c linkedInDashSocialCounts
			if err := json.Unmarshal(msg, &c); err == nil {
				countsByURN[c.EntityURN] = c
			}
		}
	}

	posts := make([]linkedInPost, 0, len(updates))
	for _, u := range updates {
		key := linkedInSocialCountsPrefix + u.Metadata.BackendURN
		c := countsByURN[key]
		posts = append(posts, linkedInPost{update: u, likes: c.NumLikes, comments: c.NumComments})
	}
	return posts, nil
}

// linkedInPost is a self-contained post with pre-joined social counts,
// implementing ingest.Transformer so it can be passed to ingest.TransformAll.
type linkedInPost struct {
	update   linkedInDashUpdate
	likes    int
	comments int
}

func (p linkedInPost) ShouldInclude() bool   { return p.likes >= linkedInMinLikes }
func (p linkedInPost) EnrichmentURL() string { return "" }

func (p linkedInPost) Transform() news.Item {
	u := p.update
	text := u.Commentary.Text.Text

	// Strip UTM and other query params from the share URL.
	shareURL := u.SocialContent.ShareURL
	if i := strings.IndexByte(shareURL, '?'); i >= 0 {
		shareURL = shareURL[:i]
	}

	// Strip miniProfileUrn and other query params from the profile URL.
	profileURL := u.Actor.NavigationContext.ActionTarget
	if i := strings.IndexByte(profileURL, '?'); i >= 0 {
		profileURL = profileURL[:i]
	}

	return news.Item{
		Source: news.SourceLinkedIn,
		Title:  linkedInTitle(text),
		URL:    shareURL,
		Author: &news.Author{
			Name:       u.Actor.Name.Text,
			Username:   u.Actor.Description.Text,
			ProfileURL: profileURL,
		},
		Snippet:  text,
		Tag:      news.TagSocial,
		Comments: p.comments,
		Score:    news.ScoreOf(news.SourceLinkedIn, news.TagSocial, float64(p.likes), true),
	}
}

// linkedInCSRFFromCookie extracts the JSESSIONID value from the cookie string.
// The csrf-token header must equal the JSESSIONID cookie value for LinkedIn's
// CSRF check to pass.
func linkedInCSRFFromCookie(cookie string) string {
	for _, part := range strings.Split(cookie, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "JSESSIONID=") {
			return strings.Trim(strings.TrimPrefix(part, "JSESSIONID="), `"`)
		}
	}
	return ""
}

var linkedInTagRe = regexp.MustCompile(`<[^>]*>`)

// linkedInTitle cleans the raw post text and returns the first sentence or
// first linkedInTitleMaxLen runes, whichever is shorter.
func linkedInTitle(text string) string {
	clean := linkedInTagRe.ReplaceAllString(text, " ")
	clean = html.UnescapeString(clean)
	if i := strings.Index(clean, "\n"); i >= 0 {
		clean = clean[:i]
	}
	clean = strings.Join(strings.Fields(clean), " ")
	if i := strings.IndexAny(clean, ".!?"); i > 0 && i < linkedInTitleMaxLen {
		return strings.TrimSpace(clean[:i])
	}
	r := []rune(clean)
	if len(r) > linkedInTitleMaxLen {
		return strings.TrimSpace(string(r[:linkedInTitleMaxLen]))
	}
	return strings.TrimSpace(clean)
}

// ---- Response types ---------------------------------------------------------

type (
	linkedInDashUpdate struct {
		EntityURN     string                    `json:"entityUrn"`
		Metadata      linkedInDashUpdateMeta    `json:"metadata"`
		Actor         linkedInDashActor         `json:"actor"`
		Commentary    linkedInDashCommentary    `json:"commentary"`
		SocialContent linkedInDashSocialContent `json:"socialContent"`
	}

	linkedInDashUpdateMeta struct {
		BackendURN string `json:"backendUrn"`
	}

	linkedInDashActor struct {
		Name struct {
			Text string `json:"text"`
		} `json:"name"`
		Description struct {
			Text string `json:"text"`
		} `json:"description"`
		NavigationContext struct {
			ActionTarget string `json:"actionTarget"`
		} `json:"navigationContext"`
	}

	linkedInDashCommentary struct {
		Text struct {
			Text string `json:"text"`
		} `json:"text"`
	}

	linkedInDashSocialContent struct {
		ShareURL string `json:"shareUrl"`
	}

	linkedInDashSocialCounts struct {
		EntityURN   string `json:"entityUrn"`
		NumLikes    int    `json:"numLikes"`
		NumComments int    `json:"numComments"`
	}
)
