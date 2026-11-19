// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metrics

import (
	"time"

	"github.com/gorilla/schema"

	"github.com/ainsleyclark/godaily/pkg/api"
)

const (
	DefaultMetricsLimit = 10
	MaxMetricsLimit     = 100
)

// decoder is the shared gorilla/schema decoder for metrics query parameters.
// IgnoreUnknownKeys is set so endpoint-specific params (bucket, metric, etc.)
// do not cause decode errors when processed by handlers that don't declare them.
var decoder = func() *schema.Decoder {
	d := schema.NewDecoder()
	d.IgnoreUnknownKeys(true)
	return d
}()

// parseDateWindow resolves optional from/to bounds; see api.ParseDateWindow.
func parseDateWindow(rawFrom, rawTo, period string) (from, to *time.Time, err error) {
	return api.ParseDateWindow(rawFrom, rawTo, period)
}
