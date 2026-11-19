// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"encoding/json"
	"html"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
)

// The fixtures in testdata/arcticshift are real, unmodified captures:
//
//   - reddit_new.json is reddit.com's own r/golang /new.json?limit=25 listing
//     (the format the legacy ScraperAPI path fetched), covering ~54 hours.
//   - window.json is Arctic Shift's /api/posts/search response for r/golang
//     over that same time window (plus the oldest listing post, which sits on
//     the exclusive "after" boundary and was fetched by ID).
//
// Together they show what changes when the source switches from reddit.com to
// Arctic Shift: which fields agree, which legitimately drift, and which posts
// Arctic Shift returns that Reddit's listing hid.

func loadRedditListingFixture(t *testing.T) []redditPost {
	t.Helper()
	b, err := os.ReadFile("testdata/arcticshift/reddit_new.json")
	require.NoError(t, err)
	var listing redditListing
	require.NoError(t, json.Unmarshal(b, &listing))
	posts := make([]redditPost, len(listing.Data.Children))
	for i, c := range listing.Data.Children {
		posts[i] = c.Data
	}
	require.Len(t, posts, 25)
	return posts
}

func loadArcticShiftWindowFixture(t *testing.T) map[string]redditPost {
	t.Helper()
	b, err := os.ReadFile("testdata/arcticshift/window.json")
	require.NoError(t, err)
	var resp arcticShiftResponse
	require.NoError(t, json.Unmarshal(b, &resp))
	require.Empty(t, resp.Error)
	posts := make(map[string]redditPost, len(resp.Data))
	for _, p := range resp.Data {
		require.NotEmpty(t, p.ID)
		posts[p.ID] = p
	}
	require.Len(t, posts, 43)
	return posts
}

// TestReddit_ArcticShiftMatchesRedditListing checks, post by post, that Arctic
// Shift's record of each of the 25 posts in a real reddit.com listing produces
// the same news.Item. Score and comment counts are excluded: each source
// snapshots them at a different moment, so they always drift.
func TestReddit_ArcticShiftMatchesRedditListing(t *testing.T) {
	t.Parallel()

	listing := loadRedditListingFixture(t)
	arctic := loadArcticShiftWindowFixture(t)

	// Posts whose body legitimately differs. Arctic Shift archives a post
	// within seconds of submission and keeps that first version; reddit.com
	// serves the current one.
	knownSnippetDiffs := map[string]string{
		"1tb077n": "author rewrote the body after Arctic Shift archived it",
		"1t9t7wa": "author appended a p.s. after Arctic Shift archived it",
		"1t9vhyn": "held by AutoModerator when archived (body \"[removed]\"), approved later",
		"1t9472r": "held by AutoModerator when archived (body \"[removed]\"), approved later",
	}
	// Posts whose preview image differs: the edit above swapped the link.
	knownImageDiffs := map[string]string{
		"1tb077n": "edited post links a different preview image",
	}

	for _, rp := range listing {
		t.Run(rp.ID, func(t *testing.T) {
			t.Parallel()

			ap, ok := arctic[rp.ID]
			require.True(t, ok, "Arctic Shift is missing a post Reddit listed")

			want := redditChild{Data: rp}.Transform()
			got := redditChild{Data: ap}.Transform()

			assert.Equal(t, want.Source, got.Source)
			assert.Equal(t, want.URL, got.URL)
			assert.Equal(t, want.OriginalURL, got.OriginalURL)
			assert.Equal(t, want.Author, got.Author)
			assert.Equal(t, want.Tag, got.Tag)
			assert.Equal(t, want.Published, got.Published)

			// Reddit HTML-escapes titles and bodies (e.g. "&lt;dialog&gt;")
			// whereas Arctic Shift stores them unescaped; TransformAll
			// unescapes both, so compare what the pipeline would emit.
			assert.Equal(t, html.UnescapeString(want.Title), html.UnescapeString(got.Title))

			if reason, ok := knownSnippetDiffs[rp.ID]; ok {
				t.Logf("snippet differs as expected: %s", reason)
			} else {
				assert.Equal(t, normaliseText(want.Snippet), normaliseText(got.Snippet))
			}

			if reason, ok := knownImageDiffs[rp.ID]; ok {
				t.Logf("image differs as expected: %s", reason)
			} else {
				assert.Equal(t, want.ImageURL, got.ImageURL)
			}

			// Both sources must agree on whether the digest keeps the post,
			// except where the archived body changes the "feedback"/"help"
			// keyword check.
			if _, edited := knownSnippetDiffs[rp.ID]; !edited {
				assert.Equal(t, redditChild{Data: rp}.ShouldInclude(), redditChild{Data: ap}.ShouldInclude())
			}
		})
	}

	// Every listing post must have been checked and every known diff must
	// still be real; a stale allowlist entry would hide a regression.
	for id := range knownSnippetDiffs {
		rp := findPost(t, listing, id)
		assert.NotEqual(t, normaliseText(rp.SelfText), normaliseText(arctic[id].SelfText), "%s is no longer a snippet diff", id)
	}
	for id := range knownImageDiffs {
		rp := findPost(t, listing, id)
		assert.NotEqual(t, redditImage(rp), redditImage(arctic[id]), "%s is no longer an image diff", id)
	}
}

// TestReddit_ArcticShiftRemovalFilter checks the removal filter against which
// posts reddit.com actually showed. Arctic Shift returns every post submitted
// in the window, including ones mods or Reddit removed, which Reddit's own
// listing hides.
func TestReddit_ArcticShiftRemovalFilter(t *testing.T) {
	t.Parallel()

	listing := loadRedditListingFixture(t)
	arctic := loadArcticShiftWindowFixture(t)

	listed := make(map[string]bool, len(listing))
	for _, p := range listing {
		listed[p.ID] = true
	}

	// Hidden by Reddit and flagged by Arctic Shift as removed or deleted.
	wantRemoved := map[string]bool{
		"1tb03aj": true, // removal_type moderator, retitled "[ Removed by moderator ]"
		"1tazaim": true, // removal_type moderator
		"1tay8cu": true, // retitled "[ Removed by moderator ]"
		"1tasu27": true, // removal_type moderator
		"1taqxtl": true, // removal_type deleted
		"1tadhov": true, // removal_type deleted
		"1tabvoy": true, // retitled "[ Removed by moderator ]"
		"1t9zrg6": true, // removal_type deleted
		"1t9o7yo": true, // removed_by_category moderator
		"1t953kc": true, // removed_by_category reddit
	}
	// Hidden by Reddit but with nothing in Arctic Shift's record that marks
	// them as removed (or only automod_filtered + "[removed]", which posts
	// Reddit did show also carry). These get through the source filter and
	// rely on downstream scoring/curation; listed so a change is noticed.
	wantLeaked := map[string]bool{
		"1ta7j5j": true, // automod_filtered, body "[removed]"
		"1t9vj4m": true, // automod_filtered, body "[removed]"
		"1t9bpm4": true, // automod_filtered, body "[removed]"
		"1t9awre": true, // automod_filtered, body "[removed]" (duplicate of 1t9vhyn)
		"1t9uwn2": true, // no markers
		"1t9nlju": true, // no markers
		"1t99kfk": true, // no markers ("feedback" in title drops it anyway)
		"1t99c1v": true, // no markers
	}

	var shownRemoved, hiddenRemoved, hiddenKept int
	for id, p := range arctic {
		removed := p.isRemoved()
		switch {
		case listed[id]:
			assert.False(t, removed, "%s was shown by Reddit but would be filtered out", id)
			if removed {
				shownRemoved++
			}
		case removed:
			assert.True(t, wantRemoved[id], "%s unexpectedly filtered", id)
			hiddenRemoved++
		default:
			assert.True(t, wantLeaked[id], "%s hidden by Reddit and not filtered", id)
			hiddenKept++
		}
	}

	assert.Equal(t, 0, shownRemoved)
	assert.Equal(t, len(wantRemoved), hiddenRemoved)
	assert.Equal(t, len(wantLeaked), hiddenKept)
	assert.Len(t, arctic, len(listing)+len(wantRemoved)+len(wantLeaked))
}

func TestRedditPost_IsRemoved(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		input redditPost
		want  bool
	}{
		"Visible post":                {input: redditPost{Title: "Go 1.27 released", Author: "gopher", SelfText: "Notes"}, want: false},
		"Automod filtered is kept":    {input: redditPost{Title: "x", RemovedByCategory: "automod_filtered"}, want: false},
		"Automod removal type kept":   {input: redditPost{Title: "x", Meta: arcticShiftMeta{RemovalType: "automod_filtered"}}, want: false},
		"Removed body alone is kept":  {input: redditPost{Title: "x", SelfText: "[removed]"}, want: false},
		"Removed by moderator":        {input: redditPost{Title: "x", RemovedByCategory: "moderator"}, want: true},
		"Removed by Reddit":           {input: redditPost{Title: "x", RemovedByCategory: "reddit"}, want: true},
		"Deleted category":            {input: redditPost{Title: "x", RemovedByCategory: "deleted"}, want: true},
		"Moderator removal type":      {input: redditPost{Title: "x", Meta: arcticShiftMeta{RemovalType: "moderator"}}, want: true},
		"Deleted later":               {input: redditPost{Title: "x", Meta: arcticShiftMeta{WasDeletedLater: true}}, want: true},
		"Removed title":               {input: redditPost{Title: "[ Removed by moderator ]"}, want: true},
		"Removed edited title":        {input: redditPost{Title: "x", Meta: arcticShiftMeta{EditedTitle: "[ Removed by Reddit ]"}}, want: true},
		"Deleted author":              {input: redditPost{Title: "x", Author: "[deleted]"}, want: true},
		"Deleted body":                {input: redditPost{Title: "x", SelfText: "[deleted]"}, want: true},
		"Removed mention in body ok":  {input: redditPost{Title: "x", SelfText: "My post got [ Removed by moderator ] once"}, want: false},
		"Removed mention in title ok": {input: redditPost{Title: "Why was my post [ Removed by moderator ]?"}, want: false},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, test.input.isRemoved())
			if test.want {
				assert.False(t, redditChild{Data: test.input}.ShouldInclude())
			}
		})
	}
}

func TestRedditSnippet(t *testing.T) {
	t.Parallel()

	assert.Empty(t, redditSnippet("[removed]"))
	assert.Empty(t, redditSnippet(" [removed]\n"))
	assert.Equal(t, "Body text", redditSnippet("Body text"))
	assert.Equal(t, "Posts get [removed] sometimes", redditSnippet("Posts get [removed] sometimes"))
	assert.Empty(t, redditChild{Data: redditPost{Title: "x", SelfText: "[removed]"}}.Transform().Snippet)
}

func TestNewReddit(t *testing.T) {
	t.Parallel()

	cfg := env.Config{ScraperAPIKeys: []string{"secret-key"}}

	t.Run("Arctic Shift by default", func(t *testing.T) {
		t.Parallel()
		r := NewReddit(cfg)
		assert.Equal(t, arcticShiftURL, r.url)
		assert.False(t, r.viaScraperAPI)
		assert.NotContains(t, r.url, "secret-key", "Arctic Shift needs no ScraperAPI key")
	})

	t.Run("Legacy ScraperAPI", func(t *testing.T) {
		t.Parallel()
		r := NewRedditScraperAPI(cfg)
		assert.True(t, r.viaScraperAPI)
		assert.True(t, strings.HasPrefix(r.url, "http://api.scraperapi.com?"))
		assert.Contains(t, r.url, "keep_headers=true")
		assert.Contains(t, r.url, "api_key=secret-key")
	})

	t.Run("Registered source uses Arctic Shift", func(t *testing.T) {
		t.Parallel()
		fetcher, err := news.Get(news.SourceReddit)
		require.NoError(t, err)
		r, ok := fetcher.(*Reddit)
		require.True(t, ok)
		assert.False(t, r.viaScraperAPI)
	})
}

func TestReddit_Fetch_ArcticShiftRequest(t *testing.T) {
	t.Parallel()

	var (
		gotPath   string
		gotQuery  map[string][]string
		gotHeader http.Header
	)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		gotHeader = r.Header.Clone()
		_, err := w.Write([]byte(`{"data":[]}`))
		assert.NoError(t, err)
	}))
	defer s.Close()

	now := time.Date(2026, 11, 19, 1, 0, 0, 0, time.UTC)
	r := Reddit{
		url: strings.Replace(arcticShiftURL, "https://arctic-shift.photon-reddit.com", s.URL, 1),
		now: func() time.Time { return now },
	}

	items, err := r.Fetch(t.Context())
	require.NoError(t, err)
	assert.Empty(t, items)

	assert.Equal(t, "/api/posts/search", gotPath)
	assert.Equal(t, []string{"golang"}, gotQuery["subreddit"])
	assert.Equal(t, []string{"desc"}, gotQuery["sort"])
	assert.Equal(t, []string{"100"}, gotQuery["limit"])
	assert.Equal(t, []string{strconv.FormatInt(now.Add(-48*time.Hour).Unix(), 10)}, gotQuery["after"])

	assert.Equal(t, arcticShiftUserAgent, gotHeader.Get("User-Agent"))
	// Reddit's browser fingerprint is only for reddit.com.
	assert.Empty(t, gotHeader.Get("Cookie"))
	assert.Empty(t, gotHeader.Get("Sec-Fetch-Mode"))
}

func TestReddit_Fetch_ArcticShift(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		status  int
		body    string
		wantErr string
		wantLen int
	}{
		"Bad Request": {
			status:  http.StatusBadRequest,
			body:    `{"data":null,"error":"'limit' must be between 1 and 100"}`,
			wantErr: "unexpected status code from reddit: 400",
		},
		"Error envelope with 200": {
			status:  http.StatusOK,
			body:    `{"data":null,"error":"Timeout. Maybe slow down a bit"}`,
			wantErr: "arctic shift: Timeout. Maybe slow down a bit",
		},
		"Malformed JSON": {
			status:  http.StatusOK,
			body:    `{"data":`,
			wantErr: "parsing response",
		},
		"Empty": {
			status:  http.StatusOK,
			body:    `{"data":[]}`,
			wantLen: 0,
		},
		"Removed posts dropped": {
			status: http.StatusOK,
			body: `{"data":[
				{"id":"a","title":"Kept post","url":"https://www.reddit.com/r/golang/comments/a/kept/","permalink":"/r/golang/comments/a/kept/","author":"gopher","selftext":"body","created_utc":1714100000,"removed_by_category":null},
				{"id":"b","title":"Spam","url":"https://www.reddit.com/r/golang/comments/b/spam/","permalink":"/r/golang/comments/b/spam/","author":"spammer","selftext":"[removed]","created_utc":1714100000,"removed_by_category":"reddit"},
				{"id":"c","title":"Gone","url":"https://www.reddit.com/r/golang/comments/c/gone/","permalink":"/r/golang/comments/c/gone/","author":"x","selftext":"","created_utc":1714100000,"removed_by_category":null,"_meta":{"removal_type":"moderator","edited_title":"[ Removed by moderator ]"}}
			]}`,
			wantLen: 1,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.status)
				_, err := w.Write([]byte(test.body))
				assert.NoError(t, err)
			}))
			defer s.Close()

			items, err := Reddit{url: s.URL}.Fetch(t.Context())
			if test.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.wantErr)
				assert.Nil(t, items)
				return
			}
			require.NoError(t, err)
			assert.Len(t, items, test.wantLen)
		})
	}
}

// TestReddit_Fetch_ArcticShiftMatchesLegacy runs the same three posts through
// the full Fetch pipeline (transform, filtering, snippet sanitising and
// enrichment) once from reddit.com's listing format and once from Arctic
// Shift's, and requires identical items apart from score and comment counts.
func TestReddit_Fetch_ArcticShiftMatchesLegacy(t *testing.T) {
	t.Parallel()

	fetch := func(t *testing.T, fixture string, viaScraperAPI bool) []news.Item {
		t.Helper()
		raw, err := os.ReadFile(fixture)
		require.NoError(t, err)

		var serverURL string
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/feed" {
				_, err := w.Write([]byte(strings.ReplaceAll(string(raw), "__SERVER_URL__", serverURL)))
				assert.NoError(t, err)
				return
			}
			// Enrichment target for link posts.
			_, err := w.Write([]byte(`<html><head><meta name="description" content="Linked page description"></head></html>`))
			assert.NoError(t, err)
		}))
		t.Cleanup(s.Close)
		serverURL = s.URL

		items, err := Reddit{url: s.URL + "/feed", viaScraperAPI: viaScraperAPI}.Fetch(t.Context())
		require.NoError(t, err)
		// Each run has its own server; restore the placeholder so the two
		// runs are comparable.
		for i := range items {
			items[i].URL = strings.ReplaceAll(items[i].URL, serverURL, "__SERVER_URL__")
		}
		return items
	}

	legacy := fetch(t, "testdata/reddit.json", true)
	arctic := fetch(t, "testdata/reddit_arcticshift.json", false)
	require.Len(t, legacy, 3)
	require.Len(t, arctic, len(legacy))

	for i := range legacy {
		want, got := legacy[i], arctic[i]
		assert.GreaterOrEqual(t, got.Comments, 0)
		assert.GreaterOrEqual(t, got.Score, 0.0)
		want.Score, got.Score = 0, 0
		want.Comments, got.Comments = 0, 0
		assert.Equal(t, want, got, "item %d (%s)", i, want.Title)
	}
}

func findPost(t *testing.T, posts []redditPost, id string) redditPost {
	t.Helper()
	for _, p := range posts {
		if p.ID == id {
			return p
		}
	}
	t.Fatalf("post %s not in fixture", id)
	return redditPost{}
}

// normaliseText unescapes HTML entities and collapses whitespace, matching
// what the snippet pipeline does before an item is stored.
func normaliseText(s string) string {
	return strings.Join(strings.Fields(html.UnescapeString(s)), " ")
}
