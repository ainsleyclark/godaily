// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package news

import (
	"context"
	"time"
)

type Fetcher interface {
	Fetch(ctx context.Context) ([]Item, error)
}

type Item struct {
	Source    Source
	Title     string
	URL       string
	Author    string
	Snippet   string
	Score     int
	Tag       Tag // source-specific hint ("proposal-accepted", "trending", "official")
	Comments  int
	Published time.Time
}

type Tag string

const (
	TagArticle  Tag = "article"
	TagProposal Tag = "proposal"
)
