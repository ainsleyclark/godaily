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
	news.Register(news.SourceGolangBridge, NewGolangBridge())
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

func (t golangBridgeTopic) shouldInclude() bool { return true }

// transform maps a golangBridgeTopic to a news.Item.
func (t golangBridgeTopic) transform() news.Item {
	return news.Item{
		Source:    news.SourceGolangBridge,
		Title:     t.Title,
		URL:       "https://forum.golangbridge.org/t/" + t.Slug + "/" + strconv.Itoa(t.ID),
		Comments:  t.PostsCount,
		Tag:       news.TagArticle,
		Published: t.CreatedAt,
	}
}

type (
	golangBridgeResponse struct {
		Users []struct {
			ID               int    `json:"id"`
			Username         string `json:"username"`
			Name             string `json:"name"`
			AvatarTemplate   string `json:"avatar_template"`
			Admin            bool   `json:"admin,omitempty"`
			TrustLevel       int    `json:"trust_level"`
			Moderator        bool   `json:"moderator,omitempty"`
			PrimaryGroupName string `json:"primary_group_name,omitempty"`
			FlairName        string `json:"flair_name,omitempty"`
			FlairGroupId     int    `json:"flair_group_id,omitempty"`
		} `json:"users"`
		PrimaryGroups []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"primary_groups"`
		FlairGroups []struct {
			ID           int    `json:"id"`
			Name         string `json:"name"`
			FlairUrl     any    `json:"flair_url"`
			FlairBgColor string `json:"flair_bg_color"`
			FlairColor   string `json:"flair_color"`
		} `json:"flair_groups"`
		TopicList struct {
			CanCreateTopic bool                `json:"can_create_topic"`
			MoreTopicsUrl  string              `json:"more_topics_url"`
			PerPage        int                 `json:"per_page"`
			Topics         []golangBridgeTopic `json:"topics"`
		} `json:"topic_list"`
	}
	golangBridgeTopic struct {
		FancyTitle         string    `json:"fancy_title"`
		ID                 int       `json:"id"`
		Title              string    `json:"title"`
		Slug               string    `json:"slug"`
		PostsCount         int       `json:"posts_count"`
		ReplyCount         int       `json:"reply_count"`
		HighestPostNumber  int       `json:"highest_post_number"`
		ImageUrl           *string   `json:"image_url"`
		CreatedAt          time.Time `json:"created_at"`
		LastPostedAt       time.Time `json:"last_posted_at"`
		Bumped             bool      `json:"bumped"`
		BumpedAt           time.Time `json:"bumped_at"`
		Archetype          string    `json:"archetype"`
		Unseen             bool      `json:"unseen"`
		Pinned             bool      `json:"pinned"`
		Unpinned           any       `json:"unpinned"`
		Excerpt            string    `json:"excerpt,omitempty"`
		Visible            bool      `json:"visible"`
		Closed             bool      `json:"closed"`
		Archived           bool      `json:"archived"`
		Bookmarked         any       `json:"bookmarked"`
		Liked              any       `json:"liked"`
		TagsDescriptions   struct{}  `json:"tags_descriptions"`
		Views              int       `json:"views"`
		LikeCount          int       `json:"like_count"`
		HasSummary         bool      `json:"has_summary"`
		LastPosterUsername string    `json:"last_poster_username"`
		CategoryId         int       `json:"category_id"`
		PinnedGlobally     bool      `json:"pinned_globally"`
		FeaturedLink       any       `json:"featured_link"`
		HasAcceptedAnswer  bool      `json:"has_accepted_answer"`
		CanVote            bool      `json:"can_vote"`
		Posters            []struct {
			Extras         *string `json:"extras"`
			Description    string  `json:"description"`
			UserId         int     `json:"user_id"`
			PrimaryGroupId *int    `json:"primary_group_id"`
			FlairGroupId   *int    `json:"flair_group_id"`
		} `json:"posters"`
	}
)
