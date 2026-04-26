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

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/ainsleyclark/godaily/internal/news"
)

// GolangBridge defines the type that implements news.Fetcher.
type GolangBridge struct {
	url string
}

var _ news.Fetcher = &GolangBridge{}

func init() {
	news.Register(news.SourceGolangBridge, func() news.Fetcher { return NewGolangBridge() })
}

const golangBridgeURL = "https://forum.golangbridge.org/latest.json"

// NewGolangBridge creates a GolangBridge Discourse forum client.
func NewGolangBridge() *GolangBridge {
	return &GolangBridge{
		url: golangBridgeURL,
	}
}

// Fetch retrieves all news items from the GolangBridge forum.
func (g GolangBridge) Fetch(ctx context.Context) ([]news.Item, error) {
	response, err := fetch[golangBridgeResponse](ctx, g.url, "golangbridge", json.Unmarshal)
	if err != nil {
		return nil, err
	}
	return transformAll(response.TopicList.Topics), nil
}

// transform maps a golangBridgeTopic to a news.Item.
func (t golangBridgeTopic) transform() news.Item {
	published, _ := time.Parse(time.RFC3339, t.CreatedAt)
	return news.Item{
		Source:    news.SourceGolangBridge,
		Title:     t.Title,
		URL:       "https://forum.golangbridge.org/t/" + t.Slug + "/" + strconv.Itoa(t.ID),
		Comments:  t.PostsCount,
		Tag:       news.TagArticle,
		Published: published,
	}
}

type (
	golangBridgeResponse struct {
		TopicList struct {
			Topics []golangBridgeTopic `json:"topics"`
		} `json:"topic_list"`
	}
	golangBridgeTopic struct {
		ID         int    `json:"id"`
		Title      string `json:"title"`
		Slug       string `json:"slug"`
		PostsCount int    `json:"posts_count"`
		CreatedAt  string `json:"created_at"`
	}
)
