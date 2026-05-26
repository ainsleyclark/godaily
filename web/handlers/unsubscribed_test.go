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

func TestUnsubscribed(t *testing.T) {
	t.Parallel()

	kit := webkit.New()
	kit.Get("/unsubscribed/", Unsubscribed())

	req := httptest.NewRequest(http.MethodGet, "/unsubscribed/", nil)
	rec := httptest.NewRecorder()
	kit.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "been unsubscribed")
}
