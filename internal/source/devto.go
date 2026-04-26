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
	"time"

	"github.com/ainsleyclark/godaily/internal/news"
)

// DevTo defines the type that implements news.Fetcher.
type DevTo struct {
	url string
}

var _ news.Fetcher = &DevTo{}

func init() {
	news.Register(news.SourceDevTo, func() news.Fetcher { return NewDevTo() })
}

const devToUrl = "https://dev.to/api/articles?tag=go&top=1"

// NewDevTo creates a dev.to client.
func NewDevTo() *DevTo {
	return &DevTo{
		url: devToUrl,
	}
}

// Fetch retrieves all the news items from dev.to
func (d DevTo) Fetch(ctx context.Context) ([]news.Item, error) {
	response, err := fetch[[]devToResponse](ctx, d.url, "dev to", json.Unmarshal)
	if err != nil {
		return nil, err
	}
	return transformAll(response), nil
}

func (d devToResponse) transform() news.Item {
	return news.Item{
		Source:    news.SourceDevTo,
		Title:     d.Title,
		URL:       d.Url,
		Author:    d.User.Name,
		Snippet:   d.Description,
		Score:     0,
		Tag:       news.TagProposal,
		Comments:  d.CommentsCount,
		Published: d.PublishedAt,
	}
}

type devToResponse struct {
	TypeOf                 string      `json:"type_of"`
	Id                     int         `json:"id"`
	Title                  string      `json:"title"`
	Description            string      `json:"description"`
	ReadablePublishDate    string      `json:"readable_publish_date"`
	Slug                   string      `json:"slug"`
	Path                   string      `json:"path"`
	Url                    string      `json:"url"`
	CommentsCount          int         `json:"comments_count"`
	PublicReactionsCount   int         `json:"public_reactions_count"`
	CollectionId           *int        `json:"collection_id"`
	PublishedTimestamp     time.Time   `json:"published_timestamp"`
	Language               string      `json:"language"`
	SubforemId             int         `json:"subforem_id"`
	PositiveReactionsCount int         `json:"positive_reactions_count"`
	CoverImage             *string     `json:"cover_image"`
	SocialImage            string      `json:"social_image"`
	CanonicalUrl           string      `json:"canonical_url"`
	CreatedAt              time.Time   `json:"created_at"`
	EditedAt               *time.Time  `json:"edited_at"`
	CrosspostedAt          interface{} `json:"crossposted_at"`
	PublishedAt            time.Time   `json:"published_at"`
	LastCommentAt          time.Time   `json:"last_comment_at"`
	ReadingTimeMinutes     int         `json:"reading_time_minutes"`
	TagList                []string    `json:"tag_list"`
	Tags                   string      `json:"tags"`
	User                   struct {
		Name            string  `json:"name"`
		Username        string  `json:"username"`
		TwitterUsername *string `json:"twitter_username"`
		GithubUsername  *string `json:"github_username"`
		UserId          int     `json:"user_id"`
		WebsiteUrl      *string `json:"website_url"`
		ProfileImage    string  `json:"profile_image"`
		ProfileImage90  string  `json:"profile_image_90"`
	} `json:"user"`
}
