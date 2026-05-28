// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metrics

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ainsleydev/webkit/pkg/webkit"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	mockengagement "github.com/ainsleyclark/godaily/pkg/mocks/engagement"
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
