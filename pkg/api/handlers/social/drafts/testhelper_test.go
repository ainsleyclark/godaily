// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drafts

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ainsleydev/webkit/pkg/webkit"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mocksocial "github.com/ainsleyclark/godaily/pkg/mocks/social"
)

// newTest constructs a per-subtest fixture. The fixture exposes the
// Handler under test, the response recorder, and the mock repository so
// each case can configure expectations on the struct fields rather than
// inside a shared closure (per pkg/api/README.md).
type testFixture struct {
	Handler  *Handler
	Recorder *httptest.ResponseRecorder
	Posts    *mocksocial.MockPostRepository
}

func newTest(t *testing.T) *testFixture {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	posts := mocksocial.NewMockPostRepository(ctrl)
	return &testFixture{
		Handler:  &Handler{socialPosts: posts},
		Recorder: httptest.NewRecorder(),
		Posts:    posts,
	}
}

// withRequest binds an HTTP request (and any chi URL params) to the
// fixture's recorder, returning the webkit.Context ready for handler
// invocation. Chi params are propagated via RouteCtxKey so c.Param("id")
// resolves the way the production router would.
func (f *testFixture) withRequest(method, path, body string, params map[string]string) *webkit.Context {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if len(params) > 0 {
		rctx := chi.NewRouteContext()
		for k, v := range params {
			rctx.URLParams.Add(k, v)
		}
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	}
	return webkit.NewContext(f.Recorder, req)
}

// assertNoErr is a tiny shim around require.NoError so each handler test
// reads as a single statement without a stray import in test files.
func assertNoErr(t *testing.T, err error) {
	t.Helper()
	require.NoError(t, err)
}
