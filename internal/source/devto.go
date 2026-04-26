package source

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/ainsleydev/webkit/pkg/util/httputil"
	"github.com/pkg/errors"
)

// DevTo defines the type that implements news.Fetcher.
type DevTo struct {
	http *http.Client
	url  string
}

var _ news.Fetcher = &DevTo{}

func init() {
	news.Register(news.SourceDevTo, func() news.Fetcher { return NewDevTo() })
}

const devToUrl = "https://dev.to/api/articles?tag=go&top=1"

// NewDevTo creates a dev.to client.
func NewDevTo() *DevTo {
	return &DevTo{
		http: &http.Client{},
		url:  devToUrl,
	}
}

// Fetch retrieves all the news items from dev.to
func (d DevTo) Fetch(ctx context.Context) ([]news.Item, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", d.url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "devto request creation failed")
	}

	resp, err := d.http.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "fetch dev to")
	}

	if !httputil.Is2xx(resp.StatusCode) {
		return nil, errors.Errorf("unexpected status code from dev.to: %d", resp.StatusCode)
	}

	var response []devToResponse
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, errors.Wrap(err, "parsing response")
	}

	out := make([]news.Item, len(response))
	for i, item := range response {
		out[i] = item.transform(ctx)
	}

	return out, nil
}

func (d devToResponse) transform(_ context.Context) news.Item {
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
