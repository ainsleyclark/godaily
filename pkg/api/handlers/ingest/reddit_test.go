// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ingest

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ainsleydev/webkit/pkg/webkit"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	mockdigest "github.com/ainsleyclark/godaily/pkg/mocks/digest"
	"github.com/ainsleyclark/godaily/pkg/mocks/slack"
)

// A minimal but valid r/golang listing payload. Uses a self-post (url points at
// reddit) so transforming it needs no network enrichment, and a clearly-English
// title so the language filter keeps it.
const redditJSON = `{"data":{"children":[
	{"data":{"title":"Understanding Go channels and goroutines","url":"https://www.reddit.com/r/golang/comments/x/understanding/","selftext":"A deep dive into concurrency","author":"gopher","score":42,"num_comments":3,"created_utc":1748000000,"permalink":"/r/golang/comments/x/understanding/"}}
]}}`

type deps struct {
	Handler  *Handler
	Context  *webkit.Context
	Recorder *httptest.ResponseRecorder
	Runner   *mockdigest.MockService
}

func setup(t *testing.T, body string) deps {
	t.Helper()
	ctrl := gomock.NewController(t)
	runner := mockdigest.NewMockService(ctrl)
	slackMock := mockslack.NewMockSender(ctrl)
	slackMock.EXPECT().MustSend(gomock.Any(), gomock.Any()).AnyTimes()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/ingest/reddit", strings.NewReader(body))
	return deps{
		Handler:  &Handler{runner: runner, slack: slackMock},
		Context:  webkit.NewContext(rec, req),
		Recorder: rec,
		Runner:   runner,
	}
}

func TestReddit(t *testing.T) {
	t.Parallel()

	t.Run("Ingests valid payload", func(t *testing.T) {
		t.Parallel()

		d := setup(t, redditJSON)
		d.Runner.EXPECT().
			Submit(gomock.Any(), news.SourceReddit, gomock.Any()).
			Return(digest.SubmitResponse{Received: 1, Persisted: 1}, nil)

		err := d.Handler.Reddit(d.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, d.Recorder.Code)
		assert.Contains(t, d.Recorder.Body.String(), "Ingested 1 new Reddit items")
	})

	t.Run("Passes parsed items through to the service", func(t *testing.T) {
		t.Parallel()

		d := setup(t, redditJSON)
		d.Runner.EXPECT().
			Submit(gomock.Any(), news.SourceReddit, gomock.Cond(func(items []news.Item) bool {
				return len(items) == 1 && items[0].Title == "Understanding Go channels and goroutines"
			})).
			Return(digest.SubmitResponse{Received: 1, Persisted: 1}, nil)

		err := d.Handler.Reddit(d.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, d.Recorder.Code)
	})

	t.Run("Empty body is rejected", func(t *testing.T) {
		t.Parallel()

		d := setup(t, "")

		err := d.Handler.Reddit(d.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, d.Recorder.Code)
	})

	t.Run("Invalid JSON is rejected", func(t *testing.T) {
		t.Parallel()

		d := setup(t, "not json")

		err := d.Handler.Reddit(d.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, d.Recorder.Code)
	})

	t.Run("All duplicates reports nothing new", func(t *testing.T) {
		t.Parallel()

		d := setup(t, redditJSON)
		d.Runner.EXPECT().
			Submit(gomock.Any(), news.SourceReddit, gomock.Any()).
			Return(digest.SubmitResponse{Received: 1, Duplicates: 1}, nil)

		err := d.Handler.Reddit(d.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, d.Recorder.Code)
		assert.Contains(t, d.Recorder.Body.String(), "already present")
	})

	t.Run("No in-window items reports window message", func(t *testing.T) {
		t.Parallel()

		d := setup(t, redditJSON)
		d.Runner.EXPECT().
			Submit(gomock.Any(), news.SourceReddit, gomock.Any()).
			Return(digest.SubmitResponse{Received: 1}, nil)

		err := d.Handler.Reddit(d.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, d.Recorder.Code)
		assert.Contains(t, d.Recorder.Body.String(), "collection window")
	})

	t.Run("Service error returns internal server error", func(t *testing.T) {
		t.Parallel()

		d := setup(t, redditJSON)
		d.Runner.EXPECT().
			Submit(gomock.Any(), news.SourceReddit, gomock.Any()).
			Return(digest.SubmitResponse{}, errors.New("boom"))

		_ = d.Handler.Reddit(d.Context)

		assert.Equal(t, http.StatusInternalServerError, d.Recorder.Code)
	})
}
