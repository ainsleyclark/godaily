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
	Submit(ctx context.Context, source news.Source, items []news.Item) (SubmitResponse, error)
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

// SubmitResponse is the result of a Submit call — manually supplying a source's
// items (e.g. raw Reddit JSON) when its live fetch is blocked.
type SubmitResponse struct {
	// Received is the number of transformed items in the submitted payload.
	Received int
	// Persisted is the number of new items that fell within the collection
	// window and were saved.
	Persisted int
	// Duplicates is the number of in-window items skipped because an item with
	// the same (url, tag) already existed — either in the store or earlier in
	// the same payload. Lets the endpoint be run repeatedly without creating
	// duplicates.
	Duplicates int
}
