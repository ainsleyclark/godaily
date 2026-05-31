// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ainsleydev/webkit/pkg/webkit"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/env"
	mockaudience "github.com/ainsleyclark/godaily/pkg/mocks/audience"
)

func TestNudge(t *testing.T) {
	t.Parallel()

	setup := func(t *testing.T) (*Handler, *webkit.Context, *httptest.ResponseRecorder, *mockaudience.MockSubscriberService) {
		t.Helper()

		ctrl := gomock.NewController(t)
		subs := mockaudience.NewMockSubscriberService(ctrl)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/digest/nudge", nil)

		h := &Handler{subscribers: subs, config: &env.Config{}}
		return h, webkit.NewContext(rec, req), rec, subs
	}

	t.Run("Sends nudges successfully", func(t *testing.T) {
		t.Parallel()

		h, ctx, rec, subs := setup(t)
		subs.EXPECT().SendConfirmationNudges(gomock.Any()).Return(3, 0, nil)

		err := h.Nudge(ctx)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Returns internal server error on failure", func(t *testing.T) {
		t.Parallel()

		h, ctx, rec, subs := setup(t)
		subs.EXPECT().SendConfirmationNudges(gomock.Any()).Return(0, 0, errors.New("boom"))

		_ = h.Nudge(ctx)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}
