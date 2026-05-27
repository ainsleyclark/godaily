// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package news

import "context"

// Fetcher defines the method for obtaining news items
// from various sources.
type Fetcher interface {
	// Fetch obtains a transforms news articles.
	//
	// Source types are responsible for returning errors
	// if they could not be obtained.
	Fetch(ctx context.Context) ([]Item, error)
}

// Source defines a provider or source of information.
type Source string

// SourceItems groups a source with its fetched news items.
type SourceItems struct {
	Source Source `json:"source"`
	Items  []Item `json:"items"`
}

// Source constants.
const (
	SourceDevTo          Source = "dev_to"
	SourceGoBlog         Source = "go_blog"
	SourceGitHub         Source = "github"
	SourceGitHubTrending Source = "github_trending"
	SourceReddit         Source = "reddit"
	SourceHN             Source = "hacker_news"
	SourceGolangBridge   Source = "golangbridge"
	SourceLobsters       Source = "lobsters"
	SourceMedium         Source = "medium"
	SourceYouTube        Source = "youtube"
	SourceGoPodcast      Source = "go_podcast"
	SourceFallthrough    Source = "fallthrough"
	SourceArdanLabs      Source = "ardanlabs_podcast"
	SourceGoRelease      Source = "go_release"
	SourceMastodon       Source = "mastodon"
	SourceAwesomeGo      Source = "awesome_go"
	SourceJetBrains      Source = "jetbrains"
	SourceGolangNuts     Source = "golang_nuts"
	SourcePlanetGolang   Source = "planet_golang"
	SourceMeetup         Source = "meetup"
	SourceConferences    Source = "conferences"
	SourceGoVuln         Source = "go_vuln"
	SourceHNJobs         Source = "hacker_news_jobs"
	SourceRemoteOK       Source = "remote_ok"
)

// Sources defines a list of all source types.
var Sources = []Source{
	SourceDevTo,
	SourceGoBlog,
	SourceGitHub,
	SourceGitHubTrending,
	SourceReddit,
	SourceHN,
	SourceGolangBridge,
	SourceLobsters,
	SourceMedium,
	SourceYouTube,
	SourceGoPodcast,
	SourceFallthrough,
	SourceArdanLabs,
	SourceGoRelease,
	SourceMastodon,
	SourceAwesomeGo,
	SourceJetBrains,
	SourceGolangNuts,
	SourcePlanetGolang,
	SourceMeetup,
	SourceConferences,
	SourceGoVuln,
	SourceHNJobs,
	SourceRemoteOK,
}

// FeaturedSources is the curated subset rendered on the marketing homepage,
// ordered for display.
var FeaturedSources = []Source{
	SourceHN,
	SourceReddit,
	SourceLobsters,
	SourceDevTo,
	SourceGitHubTrending,
	SourceGitHub,
	SourceYouTube,
	SourceGoBlog,
	SourceGoRelease,
	SourceJetBrains,
	SourceMedium,
	SourceGolangBridge,
	SourceGoPodcast,
	SourceArdanLabs,
	SourceFallthrough,
	SourceMastodon,
	SourceAwesomeGo,
	SourceGolangNuts,
	SourceMeetup,
	SourceConferences,
	SourceGoVuln,
}

// String implements fmt.Stringer on source.
func (s Source) String() string {
	return string(s)
}

var sourcePriorities = map[Source]int{
	SourceGoRelease:      20,
	SourceGoBlog:         19,
	SourceGitHub:         18,
	SourceGitHubTrending: 17,
	SourceHN:             16,
	SourceLobsters:       15,
	SourceReddit:         14,
	SourceJetBrains:      13,
	SourceDevTo:          12,
	SourceGolangBridge:   11,
	SourceConferences:    21,
	SourceMeetup:         10,
	SourceGoPodcast:      9,
	SourceFallthrough:    8,
	SourceArdanLabs:      7,
	SourceYouTube:        6,
	SourceMastodon:       5,
	SourceAwesomeGo:      4,
	SourceMedium:         3,
	SourceGolangNuts:     2,
	SourcePlanetGolang:   1,
	SourceGoVuln:         22,
	SourceHNJobs:         23,
	SourceRemoteOK:       24,
}

// Priority returns a stable per-source ordering weight, used to sort digest
// sections so authoritative sources appear above noisy ones (Go Blog at the
// top, Medium at the bottom).
func (s Source) Priority() int {
	return sourcePriorities[s]
}

var sourceNiceNames = map[Source]string{
	SourceDevTo:          "Dev.to",
	SourceGoBlog:         "Go Blog",
	SourceGitHub:         "GitHub",
	SourceGitHubTrending: "GitHub Trending",
	SourceReddit:         "Reddit",
	SourceHN:             "Hacker News",
	SourceGolangBridge:   "Golang Bridge",
	SourceLobsters:       "Lobsters",
	SourceMedium:         "Medium",
	SourceYouTube:        "YouTube",
	SourceGoPodcast:      "go podcast()",
	SourceFallthrough:    "Fallthrough",
	SourceArdanLabs:      "Ardan Labs Podcast",
	SourceGoRelease:      "Go Releases",
	SourceMastodon:       "Mastodon",
	SourceAwesomeGo:      "Awesome Go",
	SourceJetBrains:      "JetBrains GoLand",
	SourceGolangNuts:     "Golang Nuts",
	SourcePlanetGolang:   "Planet Golang",
	SourceMeetup:         "Meetup",
	SourceConferences:    "Go Conferences",
	SourceGoVuln:         "Go Vulnerabilities",
	SourceHNJobs:         "HN Who's Hiring",
	SourceRemoteOK:       "Remote OK",
}

// NiceName returns a formatted string of the source.
func (s Source) NiceName() string {
	nn, ok := sourceNiceNames[s]
	if !ok {
		return ""
	}
	return nn
}

var sourceEmojis = map[Source]string{
	SourceGoRelease:      "🚀",
	SourceGoBlog:         "📝",
	SourceGitHub:         "🐙",
	SourceGitHubTrending: "📦",
	SourceHN:             "🏆",
	SourceLobsters:       "🦞",
	SourceReddit:         "🤖",
	SourceYouTube:        "🎥",
	SourceGoPodcast:      "🎙",
	SourceArdanLabs:      "🎙",
	SourceDevTo:          "📰",
	SourceMedium:         "✍️",
	SourceJetBrains:      "🧠",
	SourceAwesomeGo:      "⭐",
	SourceMastodon:       "🐘",
	SourceGolangBridge:   "🌉",
	SourceFallthrough:    "📡",
	SourceGolangNuts:     "📬",
	SourcePlanetGolang:   "🌐",
	SourceMeetup:         "📅",
	SourceConferences:    "🎤",
	SourceGoVuln:         "🔒",
	SourceHNJobs:         "💼",
	SourceRemoteOK:       "🌍",
}

// Emoji returns the display emoji for the source.
func (s Source) Emoji() string {
	if e, ok := sourceEmojis[s]; ok {
		return e
	}
	return "📰"
}

// IsRanked reports whether items from this source should display a rank badge.
func (s Source) IsRanked() bool {
	return s == SourceHN || s == SourceLobsters || s == SourceReddit
}

var sourceMarkURLs = map[Source]string{
	SourceArdanLabs:    "/assets/images/marks/ardanlabs_podcast.svg",
	SourceDevTo:        "/assets/images/marks/dev_to.svg",
	SourceGitHub:       "/assets/images/marks/github.svg",
	SourceGoBlog:       "/assets/images/marks/go_blog.svg",
	SourceGoPodcast:    "/assets/images/marks/go_podcast.png",
	SourceGolangBridge: "/assets/images/marks/golangbridge.png",
	SourceHN:           "/assets/images/marks/hacker_news.svg",
	SourceJetBrains:    "/assets/images/marks/goland.svg",
	SourceLobsters:     "/assets/images/marks/lobsters.png",
	SourceMastodon:     "/assets/images/marks/mastodon.svg",
	SourceMedium:       "/assets/images/marks/medium.svg",
	SourceReddit:       "/assets/images/marks/reddit.svg",
	SourceYouTube:      "/assets/images/marks/youtube.svg",
	SourceGolangNuts:   "/assets/images/marks/golang_nuts.svg",
	SourceMeetup:       "/assets/images/marks/meetup.svg",
	SourceGoVuln:       "/assets/images/marks/go_vuln.svg",
}

// MarkURL returns the public path of the source's mark/logo asset, or ""
// when no mark file is registered (caller should fall back to ShortLabel).
func (s Source) MarkURL() string {
	return sourceMarkURLs[s]
}

var sourceShortLabels = map[Source]string{
	SourceArdanLabs:      "AL",
	SourceAwesomeGo:      "AG",
	SourceDevTo:          "DEV",
	SourceFallthrough:    "FT",
	SourceGitHub:         "GH",
	SourceGitHubTrending: "GH",
	SourceGoBlog:         "go",
	SourceGoPodcast:      "GP",
	SourceGoRelease:      "go",
	SourceGolangBridge:   "GB",
	SourceHN:             "HN",
	SourceJetBrains:      "JB",
	SourceLobsters:       "LO",
	SourceMastodon:       "M",
	SourceMedium:         "M",
	SourceReddit:         "r/",
	SourceYouTube:        "YT",
	SourceGolangNuts:     "GN",
	SourcePlanetGolang:   "PG",
	SourceMeetup:         "MT",
	SourceConferences:    "GC",
	SourceGoVuln:         "SEC",
	SourceHNJobs:         "HN",
	SourceRemoteOK:       "RO",
}

// ShortLabel returns the 2–3 character chip rendered when a mark is absent,
// and also used as the alt text alongside a mark.
func (s Source) ShortLabel() string {
	return sourceShortLabels[s]
}
