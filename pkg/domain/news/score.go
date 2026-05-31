// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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
	case SourceHNJobs:
		return 1.4
	case SourceRemoteOK:
		return 1.0
	case SourceMedium:
		return 0.75
	default:
		return 1.0
	}
}

// Per-job scoring constants. Jobs lack the engagement signals other sources
// rely on (points, stars, views), so ranking leans on intrinsic relevance:
// is the role actually about Go, how fresh is the listing, and how much
// information does the poster disclose.
const (
	jobGoBoost      = 1.5  // role title mentions Go / Golang as a whole word
	jobSalaryBoost  = 1.2  // salary range disclosed
	jobRemoteBoost  = 1.15 // remote-friendly
	jobRecencyDays  = 7.0  // exp decay scale: ~37% at 7 days, ~14% at 14 days
	jobRecencyFloor = 0.1  // never decay below this so a strong-on-other-axes listing isn't entirely buried
)

// JobBoost returns the multiplier applied on top of SourceWeight when ranking
// items in the Jobs section. Combines Go-in-title relevance, recency decay,
// salary disclosure, and remote-friendly status. ageDays clamps at zero so
// future-dated listings don't get a runaway boost.
func JobBoost(ageDays int, goInTitle, hasSalary, isRemote bool) float64 {
	if ageDays < 0 {
		ageDays = 0
	}
	boost := 1.0
	if goInTitle {
		boost *= jobGoBoost
	}
	if hasSalary {
		boost *= jobSalaryBoost
	}
	if isRemote {
		boost *= jobRemoteBoost
	}
	recency := math.Exp(-float64(ageDays) / jobRecencyDays)
	if recency < jobRecencyFloor {
		recency = jobRecencyFloor
	}
	return boost * recency
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
