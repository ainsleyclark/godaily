// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package news

import (
	"regexp"
	"strings"
)

// Cross-day, cross-source "already covered" de-duplication.
//
// The pipeline's primary de-dup keys on (URL, tag) within a single collection
// window — see groupIntoSections in the digest service and the
// items_url_tag_unique index. That catches the same article appearing twice in
// one window, but not the same real-world event resurfacing a day later from a
// different source: an r/golang thread about a release already covered from
// go.dev carries a different URL (the Reddit permalink) and a different tag
// (always TagDiscussion), so it slips straight past the (URL, tag) key and gets
// featured and mentioned in the intro a second time. ExcludeCovered collapses
// those against stories already shipped in recent issues, matching on URL or a
// normalised title since the URLs differ across platforms.

// textNormRe collapses any run of non-alphanumeric characters to a single space
// so punctuation and spacing differences ("Go 1.26.4 is released" vs
// "Go 1.26.4 released!") don't defeat matching.
var textNormRe = regexp.MustCompile(`[^a-z0-9]+`)

// normaliseText lowercases s and squashes punctuation/whitespace runs to a
// single space, trimming the result. It is the canonical form used to compare
// free text (titles, company names, roles) across sources.
func normaliseText(s string) string {
	s = textNormRe.ReplaceAllString(strings.ToLower(s), " ")
	return strings.TrimSpace(s)
}

// titleKey derives a cross-source de-dup key from an item's title. Returns ""
// when the title carries too little signal to match safely (fewer than three
// normalised tokens), so a thin or generic headline can't suppress a distinct
// item — a missed duplicate is preferable to dropping real news.
func titleKey(title string) string {
	norm := normaliseText(title)
	if norm == "" {
		return ""
	}
	if len(strings.Fields(norm)) < 3 {
		return ""
	}
	return norm
}

// ExcludeCovered drops items whose story already shipped in a recent issue,
// matching on canonical/original URL or a normalised title, so a cross-source
// re-post (an r/golang thread about a release already covered from go.dev) is
// not featured or mentioned a second time. The covered slice is the set of
// items linked to recently-sent issues; order of the input is preserved.
func ExcludeCovered(items, covered []Item) []Item {
	if len(covered) == 0 {
		return items
	}

	urls := make(map[string]struct{}, len(covered)*2)
	titles := make(map[string]struct{}, len(covered))
	for _, c := range covered {
		if c.URL != "" {
			urls[c.URL] = struct{}{}
		}
		if c.OriginalURL != "" {
			urls[c.OriginalURL] = struct{}{}
		}
		if k := titleKey(c.Title); k != "" {
			titles[k] = struct{}{}
		}
	}

	out := make([]Item, 0, len(items))
	for _, it := range items {
		if isCovered(it, urls, titles) {
			continue
		}
		out = append(out, it)
	}
	return out
}

// isCovered reports whether an item matches a recently-covered story by URL,
// original URL, or normalised title.
func isCovered(it Item, urls, titles map[string]struct{}) bool {
	if it.URL != "" {
		if _, ok := urls[it.URL]; ok {
			return true
		}
	}
	if it.OriginalURL != "" {
		if _, ok := urls[it.OriginalURL]; ok {
			return true
		}
	}
	if k := titleKey(it.Title); k != "" {
		if _, ok := titles[k]; ok {
			return true
		}
	}
	return false
}
