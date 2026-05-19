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
	"strings"
	"time"

	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/ingest"
	"github.com/ainsleyclark/godaily/pkg/news"
)

// YouTube defines the type that implements news.Fetcher for YouTube video search.
type YouTube struct {
	url       string
	videosURL string
	key       string
}

var _ news.Fetcher = &YouTube{}

func init() {
	news.Register(news.SourceYouTube, func(cfg env.Config) news.Fetcher { return NewYouTube(cfg) })
}

const (
	youtubeURL       = "https://www.googleapis.com/youtube/v3/search?part=snippet&q=golang&type=video&order=date&maxResults=25&relevanceLanguage=en"
	youtubeVideosURL = "https://www.googleapis.com/youtube/v3/videos?part=statistics"
)

// NewYouTube creates a YouTube client. It uses cfg.YouTubeAPIKey to authenticate
// with the YouTube Data API v3.
func NewYouTube(cfg env.Config) *YouTube {
	return &YouTube{
		url:       youtubeURL,
		videosURL: youtubeVideosURL,
		key:       cfg.YouTubeAPIKey,
	}
}

// Fetch retrieves the latest Go-related videos from YouTube, sorted by upload
// date. A second call to the videos.list endpoint enriches each result with its
// view count so items can be ranked by quality before the section limit is applied.
func (y YouTube) Fetch(ctx context.Context) ([]news.Item, error) {
	if y.key == "" {
		slog.Warn("YOUTUBE_API_KEY is not set, skipping")
		return nil, nil
	}
	sep := "?"
	if strings.Contains(y.url, "?") {
		sep = "&"
	}
	resp, err := ingest.Fetch[ytSearchResponse](ctx, y.url+sep+"key="+y.key, "youtube", json.Unmarshal)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(resp.Items))
	for _, item := range resp.Items {
		if item.ID.VideoID != "" {
			ids = append(ids, item.ID.VideoID)
		}
	}
	if len(ids) > 0 {
		statsURL := y.videosURL + "&id=" + strings.Join(ids, ",") + "&key=" + y.key
		statsResp, serr := ingest.Fetch[ytVideosResponse](ctx, statsURL, "youtube-stats", json.Unmarshal)
		if serr != nil {
			slog.WarnContext(ctx, "Failed to fetch YouTube video statistics, items will be unranked", "err", serr)
		} else {
			viewCounts := make(map[string]int64, len(statsResp.Items))
			for _, v := range statsResp.Items {
				viewCounts[v.ID] = v.Statistics.ViewCount
			}
			for i := range resp.Items {
				resp.Items[i].ViewCount = viewCounts[resp.Items[i].ID.VideoID]
			}
		}
	}

	return ingest.TransformAll(ctx, resp.Items), nil
}

func (v ytItem) ShouldInclude() bool   { return true }
func (v ytItem) EnrichmentURL() string { return "" }

func (v ytItem) Transform() news.Item {
	published, _ := time.Parse(time.RFC3339, v.Snippet.PublishedAt)
	return news.Item{
		Source:   news.SourceYouTube,
		Title:    v.Snippet.Title,
		URL:      "https://www.youtube.com/watch?v=" + v.ID.VideoID,
		ImageURL: "https://i.ytimg.com/vi/" + v.ID.VideoID + "/hqdefault.jpg",
		Author: &news.Author{
			Name:       v.Snippet.ChannelTitle,
			Username:   v.Snippet.ChannelID,
			ProfileURL: "https://www.youtube.com/channel/" + v.Snippet.ChannelID,
		},
		Snippet:   v.Snippet.Description,
		Tag:       news.TagVideo,
		Score:     news.ScoreOf(news.SourceYouTube, news.TagVideo, float64(v.ViewCount), v.ViewCount > 0),
		Published: published,
	}
}

type ytSearchResponse struct {
	Items []ytItem `json:"items"`
}

type ytItem struct {
	ID        ytID      `json:"id"`
	Snippet   ytSnippet `json:"snippet"`
	ViewCount int64     // populated after stats lookup, not from JSON
}

type ytID struct {
	VideoID string `json:"videoId"`
}

type ytSnippet struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	ChannelTitle string `json:"channelTitle"`
	ChannelID    string `json:"channelId"`
	PublishedAt  string `json:"publishedAt"`
}

type ytVideosResponse struct {
	Items []ytVideoItem `json:"items"`
}

type ytVideoItem struct {
	ID         string       `json:"id"`
	Statistics ytStatistics `json:"statistics"`
}

type ytStatistics struct {
	ViewCount int64 `json:"viewCount,string"`
}
