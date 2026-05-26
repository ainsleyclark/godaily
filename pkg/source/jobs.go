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

package source

import "regexp"

// Shared heuristics used by every TagJobs source. Kept here so HN Who-is-hiring
// and Remote OK score and filter listings the same way.

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
