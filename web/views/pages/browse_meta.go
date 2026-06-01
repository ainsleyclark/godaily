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

var tagTitles = map[news.Tag]string{
	news.TagRelease:    "Every <span class=\"listing-hero__accent\">Go release</span>, ranked.",
	news.TagProposal:   "Go <span class=\"listing-hero__accent\">proposals</span>, tracked.",
	news.TagConference: "Go <span class=\"listing-hero__accent\">conferences</span>, upcoming and past.",
	news.TagDiscussion: "Go <span class=\"listing-hero__accent\">discussions</span>, ranked by engagement.",
	news.TagEvent:      "Go <span class=\"listing-hero__accent\">events</span>, from meetups to workshops.",
	news.TagArticle:    "Go <span class=\"listing-hero__accent\">articles</span>, ranked by what people share.",
	news.TagTutorial:   "Go <span class=\"listing-hero__accent\">tutorials</span>, from beginner to expert.",
	news.TagVideo:      "Go <span class=\"listing-hero__accent\">videos</span>, talks, and screencasts.",
	news.TagTrending:   "<span class=\"listing-hero__accent\">Trending</span> Go stories, updated daily.",
	news.TagSecurity:   "Go <span class=\"listing-hero__accent\">security</span> advisories and CVEs.",
	news.TagJobs:       "Go <span class=\"listing-hero__accent\">jobs</span>, worldwide.",
}

var tagSubs = map[news.Tag]string{
	news.TagRelease:    "Every release, RC, and patch in one place. Use the filters to narrow by date or source.",
	news.TagProposal:   "Language proposals, library additions, and toolchain changes — from open to shipped.",
	news.TagConference: "Calls for papers, schedules, talk recordings, and recaps from the Go conference circuit.",
	news.TagDiscussion: "The most-upvoted Go threads from Hacker News, Reddit, and community forums.",
	news.TagEvent:      "Meetups, workshops, and community events happening around the world.",
	news.TagArticle:    "In-depth engineering posts, analysis, and community writing — ranked by what people actually share.",
	news.TagTutorial:   "Hands-on walkthroughs for all skill levels. Filter by source to find your preferred format.",
	news.TagVideo:      "Conference talks, screencasts, and video tutorials from across the Go community.",
	news.TagTrending:   "The most-shared stories across developer communities right now — updated as they surface.",
	news.TagSecurity:   "Vulnerability disclosures, CVEs, security advisories, and patched releases.",
	news.TagJobs:       "Engineering roles at companies hiring Go developers worldwide, collected daily.",
}

var tagDescs = map[news.Tag]string{
	news.TagRelease:    "Every Go release, RC, beta, and patch tracked as it lands. Ranked by community reaction and delivered in the GoDaily daily digest.",
	news.TagProposal:   "Active Go proposals from the issue tracker: new language features, standard library additions, and toolchain changes, from open to shipped.",
	news.TagConference: "GopherCon, GoLab, and every other Go conference — calls for papers, schedules, talk recordings, and recaps.",
	news.TagDiscussion: "The most-upvoted Go threads from Hacker News, Reddit, and community forums, ranked by engagement.",
	news.TagEvent:      "Go meetups, workshops, and local community events from around the world.",
	news.TagArticle:    "In-depth Go articles, engineering deep dives, and thoughtful analysis from across the community.",
	news.TagTutorial:   "Step-by-step Go tutorials, walkthroughs, and practical guides for Go developers of all skill levels.",
	news.TagVideo:      "Go conference talks, video tutorials, and screencasts from across the community.",
	news.TagTrending:   "The most-shared Go stories across Hacker News, Reddit, and social media, updated daily.",
	news.TagSecurity:   "Go vulnerability disclosures, CVEs, security advisories, and patched releases.",
	news.TagJobs:       "Go engineering roles at companies hiring worldwide, collected and ranked by GoDaily.",
}

func browseTagTitle(tag news.Tag) string {
	if s, ok := tagTitles[tag]; ok {
		return s
	}
	return tag.Title() + " — Go news."
}

func browseTagSub(tag news.Tag) string {
	if s, ok := tagSubs[tag]; ok {
		return s
	}
	return "Browse Go news filtered by " + tag.Title() + "."
}

func browseTagDesc(tag news.Tag) string {
	if s, ok := tagDescs[tag]; ok {
		return s
	}
	return "Browse Go news filtered by " + tag.Title() + ", ranked and collected by the GoDaily pipeline."
}
