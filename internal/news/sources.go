// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package news

// Source defines a provider or source of information.
type Source string

// Source constants
const (
	SourceDevTo  Source = "dev_to"
	SourceGoBlog Source = "go_blog"
	SourceGitHub Source = "github"
	SourceReddit Source = "reddit"
	SourceHN     Source = "hacker_news"
)

// Sources defines a list of all source types.
var Sources = []Source{
	SourceDevTo,
	SourceGoBlog,
	SourceGitHub,
	SourceReddit,
	SourceHN,
}

// String implements fmt.Stringer on source.
func (s Source) String() string {
	return string(s)
}
