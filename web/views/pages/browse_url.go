// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pages

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
)

// BrowseFilterState captures the user-visible filter slice the page renders
// from. It is a flat, link-friendly mirror of news.ItemListOptions — easier
// to compare for active states than the typed options struct.
type BrowseFilterState struct {
	Tab     string   // "" or "all" = no tab
	Sources []string // raw source values
	Query   string
	Sort    string // "hot" | "top" | "new"
	Range   string // "today" | "week" | "month" | "year" | "all"
	Digest  bool
	Page    int64
}

// BrowseBasePath is the route the browse page lives at. Keep in sync with
// the route registered in web/server/server.go.
const BrowseBasePath = "/browse/"

// BrowseTagURL returns the canonical path-style URL for a tag landing page,
// e.g. "/browse/releases/".
func BrowseTagURL(tag news.Tag) string {
	return BrowseBasePath + string(tag) + "/"
}

// BrowseURL builds a URL for the browse page from the given filter state.
// Pass override funcs to mutate the state for a single link.
// When the resulting state is a canonical tag landing (only Tab set, all else
// at defaults) it returns the path-style URL via BrowseTagURL.
func BrowseURL(state BrowseFilterState, overrides ...func(*BrowseFilterState)) string {
	s := state
	s.Sources = append([]string(nil), state.Sources...)
	for _, o := range overrides {
		o(&s)
	}

	if s.Tab != "" && s.Tab != "all" && len(s.Sources) == 0 && s.Query == "" &&
		(s.Sort == "" || s.Sort == string(news.ItemSortHot)) &&
		(s.Range == "" || s.Range == "week") &&
		!s.Digest && s.Page <= 1 {
		return BrowseTagURL(news.Tag(s.Tab))
	}

	v := url.Values{}
	if s.Tab != "" && s.Tab != "all" {
		v.Set("tab", s.Tab)
	}
	for _, src := range s.Sources {
		if src != "" {
			v.Add("source", src)
		}
	}
	if s.Query != "" {
		v.Set("q", s.Query)
	}
	if s.Sort != "" && s.Sort != string(news.ItemSortHot) {
		v.Set("sort", s.Sort)
	}
	if s.Range != "" && s.Range != "week" {
		v.Set("range", s.Range)
	}
	if s.Digest {
		v.Set("digest", "1")
	}
	if s.Page > 1 {
		v.Set("page", strconv.FormatInt(s.Page, 10))
	}

	if len(v) == 0 {
		return BrowseBasePath
	}
	return BrowseBasePath + "?" + v.Encode()
}

// WithTab returns an override that sets the tab and resets pagination.
func WithTab(tab string) func(*BrowseFilterState) {
	return func(s *BrowseFilterState) { s.Tab = tab; s.Page = 1 }
}

// WithSort sets the sort and resets pagination.
func WithSort(sort string) func(*BrowseFilterState) {
	return func(s *BrowseFilterState) { s.Sort = sort; s.Page = 1 }
}

// WithRange sets the range and resets pagination.
func WithRange(r string) func(*BrowseFilterState) {
	return func(s *BrowseFilterState) { s.Range = r; s.Page = 1 }
}

// WithDigest sets the digest-only flag and resets pagination.
func WithDigest(on bool) func(*BrowseFilterState) {
	return func(s *BrowseFilterState) { s.Digest = on; s.Page = 1 }
}

// WithToggleSource toggles a source in/out of the active set and resets
// pagination.
func WithToggleSource(src string) func(*BrowseFilterState) {
	return func(s *BrowseFilterState) {
		out := s.Sources[:0]
		removed := false
		for _, existing := range s.Sources {
			if existing == src {
				removed = true
				continue
			}
			out = append(out, existing)
		}
		if !removed {
			out = append(out, src)
		}
		s.Sources = out
		s.Page = 1
	}
}

// WithoutSource removes a single source from the active set.
func WithoutSource(src string) func(*BrowseFilterState) {
	return func(s *BrowseFilterState) {
		out := s.Sources[:0]
		for _, existing := range s.Sources {
			if existing == src {
				continue
			}
			out = append(out, existing)
		}
		s.Sources = out
		s.Page = 1
	}
}

// WithoutSources clears the source filter.
func WithoutSources() func(*BrowseFilterState) {
	return func(s *BrowseFilterState) { s.Sources = nil; s.Page = 1 }
}

// WithPage sets the page number.
func WithPage(page int64) func(*BrowseFilterState) {
	return func(s *BrowseFilterState) { s.Page = page }
}

// ResetAll returns an override that clears every filter.
func ResetAll() func(*BrowseFilterState) {
	return func(s *BrowseFilterState) { *s = BrowseFilterState{Sort: string(news.ItemSortHot), Range: "week"} }
}

// containsSource reports whether src is in the active set.
func containsSource(state BrowseFilterState, src string) bool {
	for _, s := range state.Sources {
		if strings.EqualFold(s, src) {
			return true
		}
	}
	return false
}
