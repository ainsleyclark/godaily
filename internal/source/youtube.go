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
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/ainsleyclark/godaily/internal/news"
)

// YouTube defines the type that implements news.Fetcher for YouTube video search.
type YouTube struct {
	url string
	key string
}

var _ news.Fetcher = &YouTube{}

func init() {
	news.Register(news.SourceYouTube, func() news.Fetcher { return NewYouTube() })
}

const youtubeURL = "https://www.googleapis.com/youtube/v3/search?part=snippet&q=golang&type=video&order=date&maxResults=25"

// NewYouTube creates a YouTube client. It reads YOUTUBE_API_KEY from the
// environment to authenticate with the YouTube Data API v3.
func NewYouTube() *YouTube {
	return &YouTube{
		url: youtubeURL,
		key: os.Getenv("YOUTUBE_API_KEY"),
	}
}

// Fetch retrieves the latest Go-related videos from YouTube, sorted by upload date.
func (y YouTube) Fetch(ctx context.Context) ([]news.Item, error) {
	if y.key == "" {
		slog.Warn("youtube: YOUTUBE_API_KEY is not set, skipping")
		return nil, nil
	}
	sep := "?"
	if strings.Contains(y.url, "?") {
		sep = "&"
	}
	resp, err := fetch[ytSearchResponse](ctx, y.url+sep+"key="+y.key, "youtube", json.Unmarshal)
	if err != nil {
		return nil, err
	}
	return transformAll(resp.Items), nil
}

func (v ytItem) transform() news.Item {
	snippet := v.Snippet.Description
	if len(snippet) > 200 {
		snippet = snippet[:200]
	}
	published, _ := time.Parse(time.RFC3339, v.Snippet.PublishedAt)
	return news.Item{
		Source:    news.SourceYouTube,
		Title:     v.Snippet.Title,
		URL:       "https://www.youtube.com/watch?v=" + v.ID.VideoID,
		Author:    v.Snippet.ChannelTitle,
		Snippet:   snippet,
		Tag:       news.TagVideo,
		Published: published,
	}
}

type ytSearchResponse struct {
	Items []ytItem `json:"items"`
}

type ytItem struct {
	ID      ytID      `json:"id"`
	Snippet ytSnippet `json:"snippet"`
}

type ytID struct {
	VideoID string `json:"videoId"`
}

type ytSnippet struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	ChannelTitle string `json:"channelTitle"`
	PublishedAt  string `json:"publishedAt"`
}
