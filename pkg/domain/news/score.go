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

import "math"

// Per-source saturation constants. A signal at or above the saturation maps to
// engagement 1.0; values are tuned against examples/raw/*.json so a top-of-the-
// day item lands near the cap rather than midway up the curve.
const (
	hnPointsSaturation            = 50.0
	redditScoreSaturation         = 100.0
	lobstersScoreSaturation       = 50.0
	devtoReactionsSaturation      = 20.0
	githubPlusOneSaturation       = 50.0
	githubTrendingStarsSaturation = 200.0
	golangBridgeViewsSaturation   = 5000.0
	mastodonFavouritesSaturation  = 20.0
	youtubeViewsSaturation        = 5000.0
	goVulnCVSSSaturation          = 10.0
)

// Engagement floors. The GitHub floor is higher so the tag-driven source weight
// always carries through — an accepted proposal with zero reactions still ranks
// well because the tag itself signals intrinsic importance.
const (
	floorDefault     = 0.1
	floorGitHub      = 0.5
	constantNoSignal = 0.5
)

// SourceWeight returns the intrinsic weight for a (source, tag) combination.
// Tag is consulted only for GitHub; other sources ignore it.
func SourceWeight(s Source, t Tag) float64 {
	switch s {
	case SourceGoRelease, SourceGoBlog, SourceGoVuln:
		return 2.0
	case SourceGitHub:
		switch t {
		case TagProposalAccepted:
			return 1.8
		case TagProposalShipped:
			return 1.7
		case TagProposal:
			// Open proposals on golang/go are curated, intentional signal —
			// they should outrank generic discussion sources (Reddit, Lobsters)
			// even with zero reactions, but stay below already-shipped proposals.
			return 1.5
		default:
			return 1.0
		}
	case SourceHN:
		return 1.2
	case SourceMedium:
		return 0.5
	default:
		return 1.0
	}
}

// LogScore maps a raw signal to [0, 1] via log(value+1)/log(saturation+1),
// flattening the long tails that engagement metrics typically follow.
func LogScore(value, saturation float64) float64 {
	if value <= 0 || saturation <= 0 {
		return 0
	}
	s := math.Log(value+1) / math.Log(saturation+1)
	if s < 0 {
		return 0
	}
	if s > 1 {
		return 1
	}
	return s
}

// ScoreOf composes the per-item score using source/tag weight and engagement.
// Sources without a numeric signal pass hasSignal=false and get a constant
// engagement of 0.5, so weight alone determines their position relative to
// other no-signal sources (e.g. Go Blog at 1.0 vs Medium at 0.25).
func ScoreOf(s Source, t Tag, signal float64, hasSignal bool) float64 {
	weight := SourceWeight(s, t)
	if !hasSignal {
		return weight * constantNoSignal
	}
	eng := LogScore(signal, saturationFor(s))
	if floor := engagementFloor(s); eng < floor {
		eng = floor
	}
	return weight * eng
}

func saturationFor(s Source) float64 {
	switch s {
	case SourceHN:
		return hnPointsSaturation
	case SourceReddit:
		return redditScoreSaturation
	case SourceLobsters:
		return lobstersScoreSaturation
	case SourceDevTo:
		return devtoReactionsSaturation
	case SourceGitHub:
		return githubPlusOneSaturation
	case SourceGitHubTrending:
		return githubTrendingStarsSaturation
	case SourceGolangBridge:
		return golangBridgeViewsSaturation
	case SourceMastodon:
		return mastodonFavouritesSaturation
	case SourceYouTube:
		return youtubeViewsSaturation
	case SourceGoVuln:
		return goVulnCVSSSaturation
	default:
		return 0
	}
}

func engagementFloor(s Source) float64 {
	if s == SourceGitHub {
		return floorGitHub
	}
	return floorDefault
}
