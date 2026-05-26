// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"context"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
)

//go:generate go run go.uber.org/mock/mockgen -package=mockdigest -destination=../../mocks/digest/Service.go . Service

// Service is the interface for the daily news aggregation pipeline.
type Service interface {
	Collect(ctx context.Context, opts CollectOptions) (CollectResponse, error)
	Build(ctx context.Context, date time.Time) error
	SendPreview(ctx context.Context, date time.Time) error
	SendDigest(ctx context.Context, date time.Time, force bool) error
	SendSuggestion(ctx context.Context, date time.Time) error
}

// CollectOptions configures a Collect call.
type CollectOptions struct {
	// DryRun skips persisting items; only the raw source items are returned.
	DryRun bool

	// Sources restricts the run to the given sources. If empty,
	// all registered sources (news.Sources) are used.
	Sources []news.Source
}

// CollectResponse is the result of a Collect call. Sources contains the
// fetched items grouped by source. Errors contains a per-source error for any
// source that failed to fetch; a source absent from Errors succeeded (even if
// it returned zero items, which is normal on quiet days).
type CollectResponse struct {
	Sources []news.SourceItems
	Errors  map[news.Source]error
}
