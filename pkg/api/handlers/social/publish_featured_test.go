// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ainsleydev/webkit/pkg/webkit"
	"github.com/stretchr/testify/assert"
)

func TestPublishFeatured(t *testing.T) {
	t.Parallel()

	t.Run("No posters configured short-circuits to OK", func(t *testing.T) {
		t.Parallel()

		h := newHandlerNoPosters(t)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/social/publish/featured", nil)
		ctx := webkit.NewContext(rec, req)

		assert.NoError(t, h.PublishFeatured(ctx))
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
