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

package social

import (
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/gateway/social"
)

// SourceProfile carries the social-media metadata for a news source — the
// per-platform mention syntax (already formatted) and a hand-written blurb
// for spotlight posts. LinkedIn entries fall back to DisplayName because
// LinkedIn's @-mention API needs URN lookups we don't do yet.
type SourceProfile struct {
	Source         news.Source
	Mentions       map[social.Platform]string
	DisplayName    string
	SpotlightBlurb string
	SourceURL      string
}

// SourceProfiles is the curated set of sources eligible for spotlight
// rotation. Adding a source is a code change so the copy stays reviewed.
//
// Mentions are best-effort: where a publisher doesn't have a handle on a
// given platform, the key is simply absent and the generator falls back
// to DisplayName.
var SourceProfiles = map[news.Source]SourceProfile{
	news.SourceArdanLabs: {
		Source:      news.SourceArdanLabs,
		DisplayName: "Ardan Labs",
		Mentions: map[social.Platform]string{
			social.PlatformBluesky:  "@ardanlabs.com",
			social.PlatformMastodon: "@ardanlabs@hachyderm.io",
		},
		SpotlightBlurb: "Bill Kennedy's writing, courses and podcast are essential Go reading. If you want depth, start here.",
		SourceURL:      "https://www.ardanlabs.com/",
	},
	news.SourceGoBlog: {
		Source:      news.SourceGoBlog,
		DisplayName: "the Go team",
		Mentions: map[social.Platform]string{
			social.PlatformBluesky:  "@golang.org",
			social.PlatformMastodon: "@golang@hachyderm.io",
		},
		SpotlightBlurb: "Release notes, design rationale and deep dives straight from the people who build Go.",
		SourceURL:      "https://go.dev/blog/",
	},
	news.SourceJetBrains: {
		Source:      news.SourceJetBrains,
		DisplayName: "JetBrains GoLand",
		Mentions: map[social.Platform]string{
			social.PlatformBluesky:  "@jetbrains.com",
			social.PlatformMastodon: "@jetbrains@mastodon.social",
		},
		SpotlightBlurb: "GoLand's blog is one of the few places consistently writing about Go tooling in depth.",
		SourceURL:      "https://blog.jetbrains.com/go/",
	},
	news.SourceGoPodcast: {
		Source:      news.SourceGoPodcast,
		DisplayName: "go podcast()",
		Mentions: map[social.Platform]string{
			social.PlatformMastodon: "@dmitshur@hachyderm.io",
		},
		SpotlightBlurb: "Short, focused interviews on what's happening in the Go ecosystem.",
		SourceURL:      "https://gopodcast.dev/",
	},
	news.SourceFallthrough: {
		Source:      news.SourceFallthrough,
		DisplayName: "Fallthrough Podcast",
		Mentions: map[social.Platform]string{
			social.PlatformBluesky: "@fallthrough.fm",
		},
		SpotlightBlurb: "The Go team's own podcast — language design discussions you won't get anywhere else.",
		SourceURL:      "https://fallthrough.fm/",
	},
	news.SourceLobsters: {
		Source:         news.SourceLobsters,
		DisplayName:    "Lobsters",
		SpotlightBlurb: "Smaller, more technical than HN. The /go feed surfaces things you'd otherwise miss.",
		SourceURL:      "https://lobste.rs/t/go",
	},
	news.SourceGoVuln: {
		Source:         news.SourceGoVuln,
		DisplayName:    "the Go security team",
		SpotlightBlurb: "If you ship Go code, you should be watching the vuln database. govulncheck is a one-liner.",
		SourceURL:      "https://pkg.go.dev/vuln/",
	},
	news.SourceAwesomeGo: {
		Source:         news.SourceAwesomeGo,
		DisplayName:    "Awesome Go",
		SpotlightBlurb: "The closest thing the Go ecosystem has to a curated package index. Worth a regular browse.",
		SourceURL:      "https://github.com/avelino/awesome-go",
	},
}

// Mention returns the platform-specific handle for a source, or the
// DisplayName when no mention is configured for that platform.
func (p SourceProfile) Mention(platform social.Platform) string {
	if m, ok := p.Mentions[platform]; ok && m != "" {
		return m
	}
	return p.DisplayName
}
