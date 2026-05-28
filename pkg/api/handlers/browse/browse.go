// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package browse serves the /browse page's filterable regions as HTML
// fragments. The page itself is generated as a static file at build time;
// listing and filtering items happens client-side by fetching these
// fragments, so the same templ components render on every surface.
package browse

import (
	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
)

// Handler holds the narrow dependencies for the browse fragment endpoint.
type Handler struct {
	issuesRepo digest.IssueRepository
	itemsRepo  news.ItemRepository
}

// New constructs a Handler from the application App.
func New(a *godaily.App) *Handler {
	return &Handler{
		issuesRepo: a.Repository.Issues,
		itemsRepo:  a.Repository.Items,
	}
}
