// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package featured

import (
	"errors"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
)

// Featured is the one item picked from the day's news to anchor every
// social post. Hook is the model's one-line reason this item matters,
// used to seed the per-platform reframing prompts.
type Featured struct {
	Title  string      `json:"title"`
	URL    string      `json:"url"`
	Source news.Source `json:"source"`
	Tag    news.Tag    `json:"tag"`
	Hook   string      `json:"hook"`
}

// ErrNoCandidates is returned by Feature when the input contains no
// items suitable for posting.
var ErrNoCandidates = errors.New("prompts: no candidate items")
