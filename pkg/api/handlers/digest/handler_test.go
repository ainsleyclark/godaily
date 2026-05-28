// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"net/http"
	"net/http/httptest"

	"github.com/ainsleydev/webkit/pkg/webkit"
)

// invoke is a legacy shim for tests not yet migrated to the
// Test struct + setup closure pattern described in pkg/api/README.md.
// Handlers now write directly to the recorder via api.OK / api.Error,
// so the only job left is to construct the context and call the handler.
func invoke(h func(*webkit.Context) error, w *httptest.ResponseRecorder, r *http.Request) {
	_ = h(webkit.NewContext(w, r))
}
