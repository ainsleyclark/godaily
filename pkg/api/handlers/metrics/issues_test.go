// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metrics

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ainsleydev/webkit/pkg/webkit"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	mockdigest "github.com/ainsleyclark/godaily/pkg/mocks/digest"
	mockengagement "github.com/ainsleyclark/godaily/pkg/mocks/engagement"
	"github.com/ainsleyclark/godaily/pkg/store"
)

func TestHandleIssues(t *testing.T) {
	t.Parallel()

	type Test struct {
		Handler  *Handler
		Context  *webkit.Context
		Recorder *httptest.ResponseRecorder
		Metrics  *mockengagement.MockMetricsRepository
	}

	setup := func(t *testing.T, query string) Test {
		t.Helper()

		ctrl := gomock.NewController(t)
		metricsMock := mockengagement.NewMockMetricsRepository(ctrl)
		rec := httptest.NewRecorder()

		target := "/metrics/issues"
		if query != "" {
			target += "?" + query
		}
		req := httptest.NewRequest(http.MethodGet, target, nil)

		return Test{
			Handler:  &Handler{metricsRepo: metricsMock},
			Context:  webkit.NewContext(rec, req),
			Recorder: rec,
			Metrics:  metricsMock,
		}
	}

	t.Run("Returns issue metrics on success", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "")
		deps.Metrics.EXPECT().IssueList(gomock.Any(), gomock.Any(), "sent_at").Return([]engagement.IssueEngagement{
			{IssueID: 1, Slug: "2026-05-22"},
		}, nil)

		err := deps.Handler.Issues(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
	})

	t.Run("Returns issue metrics with sort", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "sort=click_rate")
		deps.Metrics.EXPECT().IssueList(gomock.Any(), gomock.Any(), "click_rate").Return([]engagement.IssueEngagement{}, nil)

		err := deps.Handler.Issues(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
	})

	t.Run("Store error returns internal server error", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "")
		deps.Metrics.EXPECT().IssueList(gomock.Any(), gomock.Any(), "sent_at").Return(nil, errors.New("db error"))

		_ = deps.Handler.Issues(deps.Context)
		assert.Equal(t, http.StatusInternalServerError, deps.Recorder.Code)
	})

	t.Run("Invalid query params returns bad request", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "from=bad")

		_ = deps.Handler.Issues(deps.Context)
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})

	t.Run("Unknown sort returns bad request", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "sort=bad_sort")

		_ = deps.Handler.Issues(deps.Context)
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})
}

func TestHandleIssueTrend(t *testing.T) {
	t.Parallel()

	type Test struct {
		Handler  *Handler
		Context  *webkit.Context
		Recorder *httptest.ResponseRecorder
		Metrics  *mockengagement.MockMetricsRepository
		Issues   *mockdigest.MockIssueRepository
	}

	setup := func(t *testing.T, slug, query string) Test {
		t.Helper()

		ctrl := gomock.NewController(t)
		metricsMock := mockengagement.NewMockMetricsRepository(ctrl)
		issuesMock := mockdigest.NewMockIssueRepository(ctrl)
		rec := httptest.NewRecorder()

		target := "/metrics/issues/" + slug + "/trend"
		if query != "" {
			target += "?" + query
		}
		req := httptest.NewRequest(http.MethodGet, target, nil)
		if slug != "" {
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("slug", slug)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		}

		return Test{
			Handler:  &Handler{metricsRepo: metricsMock, issuesRepo: issuesMock},
			Context:  webkit.NewContext(rec, req),
			Recorder: rec,
			Metrics:  metricsMock,
			Issues:   issuesMock,
		}
	}

	t.Run("Returns issue trend with defaults on success", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "2026-05-22", "")
		deps.Issues.EXPECT().FindBySlug(gomock.Any(), "2026-05-22").Return(digest.Issue{ID: 1, Slug: "2026-05-22"}, nil)
		deps.Metrics.EXPECT().IssueTrend(gomock.Any(), int64(1), gomock.Any(), "unique_clicks", "hour").Return(engagement.TrendData{
			Metric: "unique_clicks",
			Bucket: "hour",
			Points: []engagement.TrendPoint{},
		}, nil)

		err := deps.Handler.IssueTrend(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
	})

	t.Run("Returns issue trend with metric and bucket", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "2026-05-22", "metric=open_rate&bucket=week")
		deps.Issues.EXPECT().FindBySlug(gomock.Any(), "2026-05-22").Return(digest.Issue{ID: 7, Slug: "2026-05-22"}, nil)
		deps.Metrics.EXPECT().IssueTrend(gomock.Any(), int64(7), gomock.Any(), "open_rate", "week").Return(engagement.TrendData{}, nil)

		err := deps.Handler.IssueTrend(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
	})

	t.Run("Missing slug returns bad request", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "", "")

		_ = deps.Handler.IssueTrend(deps.Context)
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})

	t.Run("Invalid metric returns bad request", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "2026-05-22", "metric=bad_metric")

		_ = deps.Handler.IssueTrend(deps.Context)
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})

	t.Run("Invalid bucket returns bad request", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "2026-05-22", "bucket=month")

		_ = deps.Handler.IssueTrend(deps.Context)
		assert.Equal(t, http.StatusBadRequest, deps.Recorder.Code)
	})

	t.Run("Unknown issue returns not found", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "2026-05-22", "")
		deps.Issues.EXPECT().FindBySlug(gomock.Any(), "2026-05-22").Return(digest.Issue{}, store.ErrNotFound)

		_ = deps.Handler.IssueTrend(deps.Context)
		assert.Equal(t, http.StatusNotFound, deps.Recorder.Code)
	})

	t.Run("Store error returns internal server error", func(t *testing.T) {
		t.Parallel()

		deps := setup(t, "2026-05-22", "")
		deps.Issues.EXPECT().FindBySlug(gomock.Any(), "2026-05-22").Return(digest.Issue{ID: 1, Slug: "2026-05-22"}, nil)
		deps.Metrics.EXPECT().IssueTrend(gomock.Any(), int64(1), gomock.Any(), "unique_clicks", "hour").Return(engagement.TrendData{}, errors.New("db error"))

		_ = deps.Handler.IssueTrend(deps.Context)
		assert.Equal(t, http.StatusInternalServerError, deps.Recorder.Code)
	})
}
