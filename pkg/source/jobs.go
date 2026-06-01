// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"regexp"
	"strings"
	"time"
)

// Shared heuristics used by every TagJobs source. Kept here so HN Who-is-hiring
// and Remote OK score and filter listings the same way.

// buildJobTitle composes a job link title as "Company · Role · Location",
// omitting any empty part. Putting the employer first mirrors the HN
// whoishiring convention and gives the otherwise-bare role some context. The
// "·" separator is also what the cross-source de-dup (news.jobKey) splits on to
// recover the role, so every job source should format titles this way.
func buildJobTitle(company, role, location string) string {
	parts := make([]string, 0, 3)
	for _, p := range []string{company, role, location} {
		if s := strings.TrimSpace(p); s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, " · ")
}

// goWordRe matches "go", "golang", or "go-<word>" as a whole word so listings
// such as "Mongo Atlas" or "Django" don't false-positive but "Senior Go
// engineer" and "Go-developer" do.
var goWordRe = regexp.MustCompile(`(?i)\b(?:go|golang)\b`)

// hasGoWord reports whether s mentions Go or Golang as a whole-word token.
func hasGoWord(s string) bool { return goWordRe.MatchString(s) }

// salaryRe matches currency-prefixed numbers and explicit salary phrasing.
// Used as a coarse "is the salary disclosed" signal — false positives are
// acceptable because the boost is small.
var salaryRe = regexp.MustCompile(`(?i)[$€£¥]\s*\d|\b(?:salary|comp(?:ensation)?|annual(?:ly)?|/?\s*year|equity)\b`)

// hasSalary reports whether s discloses compensation information.
func hasSalary(s string) bool { return salaryRe.MatchString(s) }

// remoteRe matches the common remote-friendly markers used by job listings.
var remoteRe = regexp.MustCompile(`(?i)\b(?:remote|worldwide|anywhere)\b`)

// isRemote reports whether s suggests the role is remote-friendly.
func isRemote(s string) bool { return remoteRe.MatchString(s) }

// jobFeedAgeDays returns whole days between an RSS pubDate and now, floored at
// zero. Unparseable or future dates yield zero so they neither error nor earn a
// runaway recency boost. Tries the timezone-offset form first, then the named
// form, covering the RFC822/RFC1123 variants feeds emit.
func jobFeedAgeDays(now time.Time, pubDate string) int {
	posted, err := time.Parse(time.RFC1123Z, pubDate)
	if err != nil {
		posted, err = time.Parse(time.RFC1123, pubDate)
		if err != nil {
			return 0
		}
	}
	days := int(now.Sub(posted.UTC()).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

// jobRoleAt splits a "<Role> @ <Company>" or "<Role> at <Company>" job title
// into its parts, trimming any trailing location/salary suffix from the company
// (" - …", " | …", " (…)"). Non-breaking spaces — which Golangprojects pads its
// "@" separator with — are normalised to regular spaces first. Returns an empty
// company and the whole title as the role when no separator is present; Go-only
// boards still rank fine without a company, they just won't take part in
// cross-source de-duplication.
func jobRoleAt(title string) (company, role string) {
	title = strings.TrimSpace(strings.ReplaceAll(title, "\u00a0", " "))
	for _, sep := range []string{" @ ", " at "} {
		i := strings.Index(title, sep)
		if i <= 0 {
			continue
		}
		role = strings.TrimSpace(title[:i])
		company = strings.TrimSpace(title[i+len(sep):])
		for _, d := range []string{" - ", " | ", " · ", " ("} {
			if j := strings.Index(company, d); j > 0 {
				company = strings.TrimSpace(company[:j])
			}
		}
		return company, role
	}
	return "", title
}
