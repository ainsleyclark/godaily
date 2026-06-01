// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package news

import (
	"regexp"
	"strings"
)

// Cross-source job de-duplication.
//
// The pipeline's primary de-dup keys on (URL, tag) — see groupIntoSections in
// the digest service and the items_url_tag_unique index. That catches the same
// article appearing twice, but not jobs: a single role is routinely cross-posted
// to Remote OK, Remotive, We Work Remotely and the Go-specific boards, each with
// a different apply URL. Those slip past the (URL, tag) key and would otherwise
// appear in the digest once per board. DedupeJobs collapses them on company +
// normalised role instead.

// jobNormRe collapses any run of non-alphanumeric characters to a single space
// so punctuation and spacing differences between boards ("Acme, Inc." vs
// "Acme Inc") don't defeat matching.
var jobNormRe = regexp.MustCompile(`[^a-z0-9]+`)

// normaliseJob lowercases s and squashes punctuation/whitespace runs to a single
// space, trimming the result. It is the canonical form used to compare company
// names and roles across sources.
func normaliseJob(s string) string {
	s = jobNormRe.ReplaceAllString(strings.ToLower(s), " ")
	return strings.TrimSpace(s)
}

// jobTitleSep splits a job title on the separators our sources use to delimit
// "Company · Role · Location" (and the HN "COMPANY | ROLE | …" convention).
func jobTitleSep(r rune) bool { return r == '·' || r == '|' }

// jobRole extracts the role from a job title. Our job sources format titles as
// "Company · Role · Location" (or the HN pipe convention), so the role is the
// first segment that isn't the company name. Falls back to the whole title when
// there are no separators.
func jobRole(title, company string) string {
	parts := strings.FieldsFunc(title, jobTitleSep)
	for _, p := range parts {
		if normaliseJob(p) == company {
			continue
		}
		return strings.TrimSpace(p)
	}
	return strings.TrimSpace(title)
}

// jobKey derives a cross-source de-dup key for a job listing from its company
// and role. Returns "" for non-job items, or when company/role is too thin to
// match safely — a missed duplicate is preferable to merging two distinct jobs.
func jobKey(i Item) string {
	if i.Tag != TagJobs {
		return ""
	}
	var company string
	if i.Author != nil {
		company = normaliseJob(i.Author.Name)
	}
	role := normaliseJob(jobRole(i.Title, company))
	// A bare company with no distinguishable role (role == company) carries no
	// signal to tell two of that company's listings apart, so don't key on it.
	if company == "" || role == "" || role == company {
		return ""
	}
	return company + "\x00" + role
}

// DedupeJobs removes cross-source duplicate job listings, keeping the
// highest-scoring instance of each company+role. Non-job items pass through
// untouched and section order/grouping is preserved. Ties keep the first
// instance encountered, which — because Collect sorts sections by source
// priority before calling this — is the higher-priority source's copy.
func DedupeJobs(sections []SourceItems) []SourceItems {
	best := make(map[string]float64)
	for _, sec := range sections {
		for _, it := range sec.Items {
			if k := jobKey(it); k != "" {
				if s, ok := best[k]; !ok || it.Score > s {
					best[k] = it.Score
				}
			}
		}
	}

	seen := make(map[string]bool)
	out := make([]SourceItems, 0, len(sections))
	for _, sec := range sections {
		kept := make([]Item, 0, len(sec.Items))
		for _, it := range sec.Items {
			if k := jobKey(it); k != "" {
				// Skip lower-scoring copies, and any copy after the best one has
				// already been kept.
				if seen[k] || it.Score < best[k] {
					continue
				}
				seen[k] = true
			}
			kept = append(kept, it)
		}
		if len(kept) > 0 {
			out = append(out, SourceItems{Source: sec.Source, Items: kept})
		}
	}
	return out
}
