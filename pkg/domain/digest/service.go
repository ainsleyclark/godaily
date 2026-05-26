// Copyright (c) 2026 godaily (Ainsley Clark)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

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
