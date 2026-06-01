// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pages

import (
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/web/views/components"
	"github.com/ainsleyclark/godaily/web/views/layouts"
)

const browseBaseURL = "https://godaily.dev"

// browsePageMeta returns PageMeta for the browse page. Clean states (no
// filters beyond an optional tab, page 1) are indexable with tag-specific
// titles and canonicals. Any other filter combination is marked noindex.
func browsePageMeta(props BrowseProps) layouts.PageMeta {
	tag := news.Tag(props.State.Tab)
	clean := browseStateIsClean(props.State)

	if props.State.Tab == "" || props.State.Tab == "all" {
		return layouts.PageMeta{
			Title:        "Browse all Go news — GoDaily",
			Description:  "The full firehose of Go news the GoDaily pipeline has collected. Filter by source, section, and date — and see which stories made the digest.",
			CanonicalURL: browseBaseURL + BrowseBasePath,
			NoIndex:      !clean,
		}
	}

	canonical := browseBaseURL + BrowseTagURL(tag)
	return layouts.PageMeta{
		Title:        tag.Title() + " — Go news, ranked daily",
		Description:  browseTagDesc(tag),
		CanonicalURL: canonical,
		NoIndex:      !clean,
	}
}

// browseHeroProps returns the ListingHero props for the browse page.
// The default (no tag) uses the original firehose copy; a tag landing
// gets a tag-specific kicker, h1, and intro paragraph.
func browseHeroProps(state BrowseFilterState) components.ListingHeroProps {
	if state.Tab == "" || state.Tab == "all" {
		return components.ListingHeroProps{
			Kicker: "Browse the archive",
			Title:  "The full firehose, with <span class=\"listing-hero__accent\">digest picks</span> marked.",
			Sub:    "Everything our pipeline collected. Use the section tabs and filters to narrow — look for the <span class=\"listing-hero__inline-mark\">In digest</span> badge to see which stories actually made it into the newsletter.",
		}
	}
	tag := news.Tag(state.Tab)
	return components.ListingHeroProps{
		Kicker: tag.Title(),
		Title:  browseTagTitle(tag),
		Sub:    browseTagSub(tag),
	}
}

// browseStateIsClean reports whether the state has no filters beyond an
// optional tab — no sources, no search query, default sort and range, page 1.
func browseStateIsClean(s BrowseFilterState) bool {
	return len(s.Sources) == 0 &&
		s.Query == "" &&
		(s.Sort == "" || s.Sort == string(news.ItemSortHot)) &&
		(s.Range == "" || s.Range == "week") &&
		!s.Digest &&
		s.Page <= 1
}

func browseTagTitle(tag news.Tag) string {
	switch tag {
	case news.TagRelease:
		return "Every <span class=\"listing-hero__accent\">Go release</span>, ranked."
	case news.TagProposal:
		return "Go <span class=\"listing-hero__accent\">proposals</span>, tracked."
	case news.TagConference:
		return "Go <span class=\"listing-hero__accent\">conferences</span>, upcoming and past."
	case news.TagDiscussion:
		return "Go <span class=\"listing-hero__accent\">discussions</span>, ranked by engagement."
	case news.TagEvent:
		return "Go <span class=\"listing-hero__accent\">events</span>, from meetups to workshops."
	case news.TagArticle:
		return "Go <span class=\"listing-hero__accent\">articles</span>, ranked by what people share."
	case news.TagTutorial:
		return "Go <span class=\"listing-hero__accent\">tutorials</span>, from beginner to expert."
	case news.TagVideo:
		return "Go <span class=\"listing-hero__accent\">videos</span>, talks, and screencasts."
	case news.TagTrending:
		return "<span class=\"listing-hero__accent\">Trending</span> Go stories, updated daily."
	case news.TagSecurity:
		return "Go <span class=\"listing-hero__accent\">security</span> advisories and CVEs."
	case news.TagJobs:
		return "Go <span class=\"listing-hero__accent\">jobs</span>, worldwide."
	default:
		return tag.Title() + " — Go news."
	}
}

func browseTagSub(tag news.Tag) string {
	switch tag {
	case news.TagRelease:
		return "Every release, RC, and patch in one place. Use the filters to narrow by date or source."
	case news.TagProposal:
		return "Language proposals, library additions, and toolchain changes — from open to shipped."
	case news.TagConference:
		return "Calls for papers, schedules, talk recordings, and recaps from the Go conference circuit."
	case news.TagDiscussion:
		return "The most-upvoted Go threads from Hacker News, Reddit, and community forums."
	case news.TagEvent:
		return "Meetups, workshops, and community events happening around the world."
	case news.TagArticle:
		return "In-depth engineering posts, analysis, and community writing — ranked by what people actually share."
	case news.TagTutorial:
		return "Hands-on walkthroughs for all skill levels. Filter by source to find your preferred format."
	case news.TagVideo:
		return "Conference talks, screencasts, and video tutorials from across the Go community."
	case news.TagTrending:
		return "The most-shared stories across developer communities right now — updated as they surface."
	case news.TagSecurity:
		return "Vulnerability disclosures, CVEs, security advisories, and patched releases."
	case news.TagJobs:
		return "Engineering roles at companies hiring Go developers worldwide, collected daily."
	default:
		return "Browse Go news filtered by " + tag.Title() + "."
	}
}

func browseTagDesc(tag news.Tag) string {
	switch tag {
	case news.TagRelease:
		return "Every Go release, RC, beta, and patch tracked as it lands. Ranked by community reaction and delivered in the GoDaily daily digest."
	case news.TagProposal:
		return "Active Go proposals from the issue tracker: new language features, standard library additions, and toolchain changes, from open to shipped."
	case news.TagConference:
		return "GopherCon, GoLab, and every other Go conference — calls for papers, schedules, talk recordings, and recaps."
	case news.TagDiscussion:
		return "The most-upvoted Go threads from Hacker News, Reddit, and community forums, ranked by engagement."
	case news.TagEvent:
		return "Go meetups, workshops, and local community events from around the world."
	case news.TagArticle:
		return "In-depth Go articles, engineering deep dives, and thoughtful analysis from across the community."
	case news.TagTutorial:
		return "Step-by-step Go tutorials, walkthroughs, and practical guides for Go developers of all skill levels."
	case news.TagVideo:
		return "Go conference talks, video tutorials, and screencasts from across the community."
	case news.TagTrending:
		return "The most-shared Go stories across Hacker News, Reddit, and social media, updated daily."
	case news.TagSecurity:
		return "Go vulnerability disclosures, CVEs, security advisories, and patched releases."
	case news.TagJobs:
		return "Go engineering roles at companies hiring worldwide, collected and ranked by GoDaily."
	default:
		return "Browse Go news filtered by " + tag.Title() + ", ranked and collected by the GoDaily pipeline."
	}
}
