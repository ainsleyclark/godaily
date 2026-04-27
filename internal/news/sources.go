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

package news

// Source defines a provider or source of information.
type Source string

// Source constants
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
}

// String implements fmt.Stringer on source.
func (s Source) String() string {
	return string(s)
}

var sourcePriorities = map[Source]int{
	SourceGoBlog:         13,
	SourceGitHub:         12,
	SourceGitHubTrending: 11,
	SourceHN:             10,
	SourceLobsters:       9,
	SourceReddit:         8,
	SourceDevTo:          7,
	SourceGolangBridge:   6,
	SourceGoPodcast:      5,
	SourceFallthrough:    4,
	SourceArdanLabs:      3,
	SourceYouTube:        2,
	SourceMedium:         1,
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
}

// NiceName returns a formatted string of the source.
func (s Source) NiceName() string {
	nn, ok := sourceNiceNames[s]
	if !ok {
		return ""
	}
	return nn
}
