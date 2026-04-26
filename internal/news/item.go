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
