// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mcpserver

import (
	"context"
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/store"
)

type (
	// GetItemInput is the input for get_item.
	GetItemInput struct {
		ID int64 `json:"id" jsonschema:"Numeric news item ID."`
	}

	// SearchItemsInput is the input for search_items. The embedded Window
	// filters on the item's published time.
	SearchItemsInput struct {
		Window
		Query    string   `json:"query,omitempty"     jsonschema:"Case-insensitive text matched against item titles and snippets."`
		Sources  []string `json:"sources,omitempty"   jsonschema:"Only items from any of these sources, e.g. hacker_news or reddit. See list_item_facets."`
		Tags     []string `json:"tags,omitempty"      jsonschema:"Only items with any of these tags, e.g. trending or official. See list_item_facets."`
		Sort     string   `json:"sort,omitempty"      jsonschema:"Ordering: new (most recently published), top (highest score) or hot (score with recency decay). Defaults to new."`
		InDigest *bool    `json:"in_digest,omitempty" jsonschema:"true for only items that made it into an issue, false for only items that did not. Omit for both."`
		Page     int64    `json:"page,omitempty"      jsonschema:"1-based page number. Defaults to 1."`
		PerPage  int64    `json:"per_page,omitempty"  jsonschema:"Items per page, at most 100. Defaults to 20."`
	}
	// SearchItemsOutput is the output for search_items.
	SearchItemsOutput struct {
		Items   []news.Item `json:"items"`
		Page    int64       `json:"page"`
		PerPage int64       `json:"per_page"`
		Total   int64       `json:"total"`
	}

	// ItemFacetsOutput is the output for list_item_facets.
	ItemFacetsOutput struct {
		TotalItems int64              `json:"total_items"`
		Sources    []news.SourceCount `json:"sources"`
		Tags       []news.TagCount    `json:"tags"`
	}
)

func (t *tools) registerItems(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_item",
		Description: "Fetch a single collected news item by ID: title, URL, source, tag, author, snippet, score and whether it was included in a digest.",
		Annotations: readOnly("Get news item"),
	}, t.getItem)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "search_items",
		Description: "Search every news item GoDaily has collected, including ones that never made it into an issue. Filter by text, source, tag, published date window and digest inclusion.",
		InputSchema: inputSchema[SearchItemsInput](withEnums(windowEnums, map[string][]string{
			"sort": {string(news.ItemSortNew), string(news.ItemSortTop), string(news.ItemSortHot)},
		})),
		Annotations: readOnly("Search news items"),
	}, t.searchItems)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_item_facets",
		Description: "List every source and tag with the number of items collected for each. Use it to discover valid values for search_items filters.",
		Annotations: readOnly("List item sources and tags"),
	}, t.itemFacets)
}

func (t *tools) getItem(ctx context.Context, _ *mcp.CallToolRequest, in GetItemInput) (*mcp.CallToolResult, news.Item, error) {
	if in.ID < 1 {
		return nil, news.Item{}, errors.New("id must be a positive integer")
	}
	item, err := t.deps.Items.Find(ctx, in.ID)
	if errors.Is(err, store.ErrNotFound) {
		return nil, news.Item{}, fmt.Errorf("item %d not found", in.ID)
	}
	if err != nil {
		return nil, news.Item{}, internalError(ctx, "fetch item", err)
	}
	return nil, item, nil
}

func (t *tools) searchItems(ctx context.Context, _ *mcp.CallToolRequest, in SearchItemsInput) (*mcp.CallToolResult, SearchItemsOutput, error) {
	from, to, err := in.bounds()
	if err != nil {
		return nil, SearchItemsOutput{}, err
	}
	page, perPage := pagination(in.Page, in.PerPage)

	opts := news.ItemListOptions{
		From:     from,
		To:       to,
		Search:   in.Query,
		Sort:     news.ItemSort(in.Sort),
		InDigest: in.InDigest,
		Page:     page,
		PerPage:  perPage,
	}
	for _, s := range in.Sources {
		opts.Sources = append(opts.Sources, news.Source(s))
	}
	for _, tag := range in.Tags {
		opts.Tags = append(opts.Tags, news.Tag(tag))
	}

	total, err := t.deps.Items.CountMatching(ctx, opts)
	if err != nil {
		return nil, SearchItemsOutput{}, internalError(ctx, "count items", err)
	}
	items, err := t.deps.Items.List(ctx, opts)
	if err != nil {
		return nil, SearchItemsOutput{}, internalError(ctx, "search items", err)
	}

	return nil, SearchItemsOutput{
		Items:   items,
		Page:    page,
		PerPage: perPage,
		Total:   total,
	}, nil
}

func (t *tools) itemFacets(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, ItemFacetsOutput, error) {
	total, err := t.deps.Items.Count(ctx)
	if err != nil {
		return nil, ItemFacetsOutput{}, internalError(ctx, "count items", err)
	}
	sources, err := t.deps.Items.SourceCounts(ctx)
	if err != nil {
		return nil, ItemFacetsOutput{}, internalError(ctx, "count items by source", err)
	}
	tags, err := t.deps.Items.TagCounts(ctx)
	if err != nil {
		return nil, ItemFacetsOutput{}, internalError(ctx, "count items by tag", err)
	}
	return nil, ItemFacetsOutput{TotalItems: total, Sources: sources, Tags: tags}, nil
}
