package source

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/pkg/errors"
)

type DevTo struct {
	client *http.Client
}

func NewDevTo() *DevTo {
	return &DevTo{
		client: &http.Client{},
	}
}

func (d *DevTo) Name() string {
	return "devto"
}

func (d *DevTo) Fetch(ctx context.Context) ([]news.Item, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://devto.me/", nil)
	if err != nil {
		return nil, errors.Wrap(err, "devto request creation failed")
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "fetch dev to")
	}

	var response []devToResponse
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, errors.Wrap(err, "parse dev to")
	}

	fmt.Print(response)

	return []news.Item{}, nil
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
