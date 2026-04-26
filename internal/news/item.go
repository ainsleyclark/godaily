package news

import (
	"context"
	"time"
)

type Source interface {
	Name() string // "reddit", "hn", …
	Fetch(ctx context.Context) ([]Item, error)
}

type Item struct {
	Source    string
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
	TagProposal Tag = "proposal"
)
