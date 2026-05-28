// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ainsleydev/webkit/pkg/webkit"
	"github.com/stretchr/testify/assert"
)

func TestHealthZ(t *testing.T) {
	t.Parallel()

	type Test struct {
		Context  *webkit.Context
		Recorder *httptest.ResponseRecorder
	}

	setup := func(t *testing.T) Test {
		t.Helper()

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		return Test{
			Context:  webkit.NewContext(rec, req),
			Recorder: rec,
		}
	}

	t.Run("Returns OK", func(t *testing.T) {
		t.Parallel()

		deps := setup(t)

		err := HealthZ(deps.Context)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, deps.Recorder.Code)
	})
}
