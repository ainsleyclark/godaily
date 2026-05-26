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

func TestPrivacy(t *testing.T) {
	t.Parallel()

	kit := webkit.New()
	kit.Get("/privacy/", Privacy())

	req := httptest.NewRequest(http.MethodGet, "/privacy/", nil)
	rec := httptest.NewRecorder()
	kit.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Privacy Policy")
}
