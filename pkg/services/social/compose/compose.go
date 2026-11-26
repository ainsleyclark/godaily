// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package compose holds the platform-aware text rules shared by every
// social post generator: how long a post really is on each platform, how
// many characters each platform allows, and the UTM-tagged footer that
// links every post back to godaily.dev.
//
// Length is measured the way each platform measures it, not as raw runes.
// Mastodon counts every URL as 23 characters and Bluesky renders URLs
// shortened (see ShortenURL), so a raw rune count overestimates long-link
// posts and caused recaps to be cut off with "…" even when they would
// have fit.
package compose

import (
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/util/aiutil"
	"github.com/ainsleyclark/godaily/pkg/utm"
)

const (
	// mastodonURLLength is the fixed weight Mastodon assigns to every URL
	// when counting a status against its 500-character limit.
	mastodonURLLength = 23
	// blueskyDisplayMax caps how many runes of a URL Bluesky shows. The
	// link facet still carries the full URL.
	blueskyDisplayMax = 30
)

// limits are the per-platform caps the generators target. Bluesky's 300
// is a hard API limit (graphemes); Mastodon's 500 is the instance
// default; LinkedIn allows 3000 but long posts get folded, so the
// generators hold themselves to 1300.
var limits = map[social.Platform]int{
	social.Bluesky:  300,
	social.LinkedIn: 1300,
	social.Mastodon: 500,
}

// URLPattern matches http/https URLs in post text. Trailing punctuation is
// stripped separately by TrimURL so it isn't treated as part of the link.
var URLPattern = regexp.MustCompile(`https?://[^\s]+`)

// TrimURL strips trailing punctuation that the URL pattern greedily
// captured but which belongs to the surrounding sentence.
func TrimURL(raw string) string {
	return strings.TrimRight(raw, ".,;:!?)'\"")
}

// Limit returns the character limit for p, or 0 when p is unknown.
func Limit(p social.Platform) int {
	return limits[p]
}

// Length reports how many characters text occupies on platform p.
func Length(p social.Platform, text string) int {
	switch p {
	case social.Mastodon:
		return lengthWithURLs(text, func(string) int { return mastodonURLLength })
	case social.Bluesky:
		return lengthWithURLs(text, func(u string) int { return utf8.RuneCountInString(ShortenURL(u)) })
	default:
		return utf8.RuneCountInString(text)
	}
}

// lengthWithURLs counts runes in text, replacing each URL's own length
// with urlLen(url).
func lengthWithURLs(text string, urlLen func(string) int) int {
	n := 0
	last := 0
	for _, m := range URLPattern.FindAllStringIndex(text, -1) {
		raw := text[m[0]:m[1]]
		uri := TrimURL(raw)
		n += utf8.RuneCountInString(text[last:m[0]])
		n += urlLen(uri)
		n += utf8.RuneCountInString(raw[len(uri):])
		last = m[1]
	}
	return n + utf8.RuneCountInString(text[last:])
}

// ShortenURL returns the display form of a URL as Bluesky shows it: no
// scheme, no "www.", no utm_* parameters and no trailing slash, capped at
// 30 runes with a trailing ellipsis. The link facet keeps the full URL,
// so a shortened display costs nothing in click-through.
func ShortenURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return capRunes(raw, blueskyDisplayMax)
	}

	q := u.Query()
	for key := range q {
		if strings.HasPrefix(key, "utm_") {
			q.Del(key)
		}
	}

	display := strings.TrimPrefix(u.Host, "www.") + u.EscapedPath()
	display = strings.TrimSuffix(display, "/")
	if enc := q.Encode(); enc != "" {
		display += "?" + enc
	}
	return capRunes(display, blueskyDisplayMax)
}

func capRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n-1]) + "…"
}

// SubscribeLine returns the "Subscribe: <url>" footer for p, tagged with
// UTM parameters so Plausible can attribute sign-ups to the platform and
// post kind (campaign) that brought them in.
func SubscribeLine(p social.Platform, campaign string) string {
	return "Subscribe: " + utm.Tag(env.AppURL+"/", "social-"+string(p), "social", campaign)
}

// WithFooter joins body and footer with a blank line, truncating body
// (never the footer) until the result fits within p's limit. Truncation
// prefers the last line break that leaves enough room, so a post loses
// whole lines rather than stopping mid-sentence. Returns body unchanged
// when footer is empty.
func WithFooter(p social.Platform, body, footer string) string {
	body = strings.TrimSpace(body)
	if footer == "" {
		return Fit(p, body)
	}
	limit := Limit(p)
	if limit == 0 {
		return body + "\n\n" + footer
	}
	budget := limit - Length(p, footer) - 2
	if budget <= 0 {
		return Fit(p, body)
	}
	return FitTo(p, body, budget) + "\n\n" + footer
}

// Fit truncates text so it fits within p's limit. See FitTo.
func Fit(p social.Platform, text string) string {
	limit := Limit(p)
	if limit == 0 {
		return text
	}
	return FitTo(p, text, limit)
}

// FitTo truncates text so Length(p, text) <= budget. It first drops
// trailing lines (keeping at least one); only when a single line is still
// too long does it fall back to a word-boundary cut with an ellipsis.
func FitTo(p social.Platform, text string, budget int) string {
	text = strings.TrimSpace(text)
	if Length(p, text) <= budget {
		return text
	}

	// Drop whole lines from the end first.
	lines := strings.Split(text, "\n")
	for len(lines) > 1 {
		lines = lines[:len(lines)-1]
		candidate := strings.TrimRightFunc(strings.Join(lines, "\n"), unicode.IsSpace)
		if candidate == "" {
			continue
		}
		if Length(p, candidate) <= budget {
			return candidate
		}
	}

	// A single over-long line: cut at a word boundary. Length can be
	// smaller than the rune count (URL weighting), so shrink the rune cap
	// until the platform length fits.
	cut := strings.TrimSpace(lines[0])
	runes := utf8.RuneCountInString(cut)
	for runes > 1 {
		out := aiutil.TruncatePost(cut, runes)
		over := Length(p, out) - budget
		if over <= 0 {
			return out
		}
		runes -= over
	}
	return ""
}
