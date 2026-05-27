// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import "github.com/ainsleyclark/godaily/pkg/domain/news"

// Profile carries the social-media metadata for a news source. It is
// the source-of-truth for spotlight posts (which need a hand-written blurb
// and the right per-platform mention syntax) and new-source announcements
// (which need a display name and source URL). Mentions keys are platform
// strings ("bluesky", "linkedin", "mastodon") rather than typed Platforms
// to keep the social package free of any gateway dependency.
type Profile struct {
	Source         news.Source
	DisplayName    string
	SourceURL      string
	SpotlightBlurb string
	Mentions       map[string]string
	// Announceable controls whether NewSource posts about this source.
	// Aggregator / community sources (HN, Reddit, Mastodon) leave this
	// false because there's nothing distinctive to shout about — they
	// just feed items into the same digest as everything else.
	Announceable bool
}

// Mention returns the platform-specific handle for a source, or the
// DisplayName when no mention is configured for that platform.
func (p Profile) Mention(platform string) string {
	if m, ok := p.Mentions[platform]; ok && m != "" {
		return m
	}
	return p.DisplayName
}

// linkedInURNKey is the Mentions map key used for LinkedIn organisation
// URNs. It is deliberately namespaced away from "linkedin" because the
// URN is out-of-band metadata for the API call — not a textual handle to
// be inlined into post copy. Keeping it under its own key means Mention
// ("linkedin") continues to fall back to DisplayName for prompts.
const linkedInURNKey = "linkedin_urn"

// LinkedInURN returns the LinkedIn organisation URN configured for this
// profile, or "" when none is set. Used by the LinkedIn platform to build
// an inline organisation mention on the post.
func (p Profile) LinkedInURN() string {
	return p.Mentions[linkedInURNKey]
}

// Profiles is the curated metadata for every source GoDaily knows about.
// Adding a row here makes that source eligible for spotlights, and (if
// Announceable) for new-source announcements when first added.
//
// Sources not in this map are skipped silently — useful for aggregator
// sources where there's no creator to tag and nothing distinctive to say.
var Profiles = map[news.Source]Profile{
	news.SourceArdanLabs: {
		Source:      news.SourceArdanLabs,
		DisplayName: "Ardan Labs",
		Mentions: map[string]string{
			"bluesky":  "@ardanlabs.com",
			"mastodon": "@ardanlabs@hachyderm.io",
		},
		SpotlightBlurb: "Bill Kennedy's writing, courses and podcast are essential Go reading. If you want depth, start here.",
		SourceURL:      "https://www.ardanlabs.com/",
		Announceable:   true,
	},
	news.SourceGoBlog: {
		Source:      news.SourceGoBlog,
		DisplayName: "the Go team",
		Mentions: map[string]string{
			"bluesky":  "@golang.org",
			"mastodon": "@golang@hachyderm.io",
		},
		SpotlightBlurb: "Release notes, design rationale and deep dives straight from the people who build Go.",
		SourceURL:      "https://go.dev/blog/",
		Announceable:   true,
	},
	news.SourceJetBrains: {
		Source:      news.SourceJetBrains,
		DisplayName: "JetBrains GoLand",
		Mentions: map[string]string{
			"bluesky":  "@jetbrains.com",
			"mastodon": "@jetbrains@mastodon.social",
		},
		SpotlightBlurb: "GoLand's blog is one of the few places consistently writing about Go tooling in depth.",
		SourceURL:      "https://blog.jetbrains.com/go/",
		Announceable:   true,
	},
	news.SourceGoPodcast: {
		Source:      news.SourceGoPodcast,
		DisplayName: "go podcast()",
		Mentions: map[string]string{
			"mastodon": "@dmitshur@hachyderm.io",
		},
		SpotlightBlurb: "Short, focused interviews on what's happening in the Go ecosystem.",
		SourceURL:      "https://gopodcast.dev/",
		Announceable:   true,
	},
	news.SourceFallthrough: {
		Source:      news.SourceFallthrough,
		DisplayName: "Fallthrough Podcast",
		Mentions: map[string]string{
			"bluesky": "@fallthrough.fm",
		},
		SpotlightBlurb: "The Go team's own podcast — language design discussions you won't get anywhere else.",
		SourceURL:      "https://fallthrough.fm/",
		Announceable:   true,
	},
	news.SourceLobsters: {
		Source:         news.SourceLobsters,
		DisplayName:    "Lobsters",
		SpotlightBlurb: "Smaller, more technical than HN. The /go feed surfaces things you'd otherwise miss.",
		SourceURL:      "https://lobste.rs/t/go",
		Announceable:   true,
	},
	news.SourceGoVuln: {
		Source:         news.SourceGoVuln,
		DisplayName:    "the Go security team",
		SpotlightBlurb: "If you ship Go code, you should be watching the vuln database. govulncheck is a one-liner.",
		SourceURL:      "https://pkg.go.dev/vuln/",
		Announceable:   true,
	},
	news.SourceAwesomeGo: {
		Source:         news.SourceAwesomeGo,
		DisplayName:    "Awesome Go",
		SpotlightBlurb: "The closest thing the Go ecosystem has to a curated package index. Worth a regular browse.",
		SourceURL:      "https://github.com/avelino/awesome-go",
		Announceable:   true,
	},
	news.SourceGoRelease: {
		Source:         news.SourceGoRelease,
		DisplayName:    "Go Releases",
		SpotlightBlurb: "Every stable, RC and beta release of the Go toolchain, pulled straight from the source.",
		SourceURL:      "https://go.dev/doc/devel/release",
		Announceable:   true,
	},
	news.SourceDevTo: {
		Source:      news.SourceDevTo,
		DisplayName: "DEV's #go community",
		Mentions: map[string]string{
			"bluesky":  "@thepracticaldev.bsky.social",
			"mastodon": "@thepracticaldev@mas.to",
		},
		SpotlightBlurb: "DEV's #go tag is one of the friendliest places to read and write about Go publicly.",
		SourceURL:      "https://dev.to/t/go",
		Announceable:   true,
	},
	news.SourceGitHubTrending: {
		Source:         news.SourceGitHubTrending,
		DisplayName:    "GitHub Trending (Go)",
		SpotlightBlurb: "The fastest-growing Go repos on GitHub — a useful pulse on what the community is building.",
		SourceURL:      "https://github.com/trending/go",
		Announceable:   true,
	},
	news.SourceGitHub: {
		Source:         news.SourceGitHub,
		DisplayName:    "the Go proposals tracker",
		SpotlightBlurb: "Every active language proposal — what's being argued about and what's about to ship.",
		SourceURL:      "https://github.com/golang/go/issues?q=is:issue+label:Proposal",
		Announceable:   true,
	},
	news.SourceConferences: {
		Source:         news.SourceConferences,
		DisplayName:    "Go Conferences",
		SpotlightBlurb: "Upcoming Go conferences worldwide, surfaced as they get close so you don't miss tickets.",
		SourceURL:      "https://go.dev/wiki/Conferences",
		Announceable:   true,
	},
	news.SourceMeetup: {
		Source:         news.SourceMeetup,
		DisplayName:    "Go Meetups",
		SpotlightBlurb: "Local Go user groups on Meetup — the best way to find Gophers near you.",
		SourceURL:      "https://www.meetup.com/topics/golang/",
		Announceable:   true,
	},
	news.SourceGolangBridge: {
		Source:         news.SourceGolangBridge,
		DisplayName:    "GolangBridge",
		SpotlightBlurb: "A long-running Go Q&A forum — slower than Discord, more searchable than Reddit.",
		SourceURL:      "https://forum.golangbridge.org/",
		Announceable:   true,
	},
	news.SourceYouTube: {
		Source:         news.SourceYouTube,
		DisplayName:    "Go talks on YouTube",
		SpotlightBlurb: "Curated channel of GopherCon, GoLab and community talks — depth beyond a blog post.",
		SourceURL:      "https://www.youtube.com/results?search_query=golang",
		Announceable:   true,
	},
}

// ProfileFor returns the profile for a source, or the zero value when no
// profile is registered. Use the boolean to distinguish "missing" from
// "registered but minimal".
func ProfileFor(s news.Source) (Profile, bool) {
	p, ok := Profiles[s]
	return p, ok
}
