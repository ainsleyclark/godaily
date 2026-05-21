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
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMeetup_Fetch(t *testing.T) {
	t.Parallel()

	fixture, err := os.ReadFile("testdata/meetup.html")
	require.NoError(t, err)

	tt := map[string]struct {
		stub func(serverURL string) http.HandlerFunc
		want func(t *testing.T, items []news.Item, err error, serverURL string)
	}{
		"Bad Request": {
			stub: func(string) http.HandlerFunc {
				return func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusBadRequest)
				}
			},
			want: func(t *testing.T, items []news.Item, err error, _ string) {
				t.Helper()
				// Failed groups are skipped; Fetch returns empty slice without error.
				assert.NoError(t, err)
				assert.Empty(t, items)
			},
		},
		"OK": {
			stub: func(string) http.HandlerFunc {
				return func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "text/html; charset=utf-8")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write(fixture)
				}
			},
			want: func(t *testing.T, items []news.Item, err error, _ string) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, items, 1)
				first := items[0]
				assert.Equal(t, "May Gophers @ Muzz!", first.Title)
				assert.Equal(t, "https://www.meetup.com/londongophers/events/314847774/", first.URL)
				assert.Equal(t, news.SourceMeetup, first.Source)
				assert.Equal(t, news.TagEvent, first.Tag)
				assert.Contains(t, first.Snippet, "London, GB")
				assert.Contains(t, first.Snippet, "80 RSVPs")
				assert.Equal(t, "https://secure.meetupstatic.com/photos/event/5/b/0/f/highres_511523311.jpeg", first.ImageURL)
			},
		},
		"No __NEXT_DATA__": {
			stub: func(string) http.HandlerFunc {
				return func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "text/html")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`<!DOCTYPE html><html><body><div id="__next"></div></body></html>`))
				}
			},
			want: func(t *testing.T, items []news.Item, err error, _ string) {
				t.Helper()
				// Missing script tag: group skipped, empty result.
				assert.NoError(t, err)
				assert.Empty(t, items)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var serverURL string
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				test.stub(serverURL)(w, r)
			}))
			defer s.Close()
			serverURL = s.URL

			got, err := (&Meetup{groupURLs: []string{s.URL + "/"}}).Fetch(t.Context())
			test.want(t, got, err, s.URL)
		})
	}
}

func TestMeetupEventItem_ShouldInclude(t *testing.T) {
	t.Parallel()

	assert.True(t, meetupEventItem{evt: meetupEvent{Status: "ACTIVE"}}.ShouldInclude())
	assert.False(t, meetupEventItem{evt: meetupEvent{Status: "PAST"}}.ShouldInclude())
	assert.False(t, meetupEventItem{evt: meetupEvent{Status: "DRAFT"}}.ShouldInclude())
	assert.False(t, meetupEventItem{evt: meetupEvent{Status: "ACTIVE", Title: "[Outside Event] AI Summit 2026"}}.ShouldInclude())
}

func TestMeetupEventItem_Transform(t *testing.T) {
	t.Parallel()

	item := meetupEventItem{
		evt: meetupEvent{
			Title:    "Go Meetup",
			EventURL: "https://www.meetup.com/londongophers/events/123/",
			Status:   "ACTIVE",
			Going:    struct{ TotalCount int `json:"totalCount"` }{TotalCount: 42},
		},
		venue: meetupVenue{City: "Berlin", Country: "de"},
		photo: meetupPhotoInfo{HighResURL: "https://example.com/photo.jpg"},
	}

	got := item.Transform()

	assert.Equal(t, news.SourceMeetup, got.Source)
	assert.Equal(t, "Go Meetup", got.Title)
	assert.Equal(t, "https://www.meetup.com/londongophers/events/123/", got.URL)
	assert.Equal(t, "https://example.com/photo.jpg", got.ImageURL)
	assert.Equal(t, news.TagEvent, got.Tag)
	assert.Contains(t, got.Snippet, "Berlin, DE")
	assert.Contains(t, got.Snippet, "42 RSVPs")
}

func TestMeetupEventItem_Transform_Online(t *testing.T) {
	t.Parallel()

	item := meetupEventItem{
		evt: meetupEvent{
			Title:    "Online Gophers",
			EventURL: "https://www.meetup.com/group/events/456/",
			Status:   "ACTIVE",
			IsOnline: true,
		},
	}
	got := item.Transform()
	assert.Contains(t, got.Snippet, "Online")
}
