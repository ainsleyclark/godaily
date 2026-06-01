// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"fmt"
	"log/slog"
	"net/http"
	"sort"

	"github.com/ainsleydev/webkit/pkg/webkit"
	slackgo "github.com/slack-go/slack"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/gateway/hook"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
	// Register all news-source fetchers (lingua-go + scrapers) so the
	// registry populates in this single binary.
	_ "github.com/ainsleyclark/godaily/pkg/source"
)

// Collect godoc
//
//	@Summary		Run the news collection pipeline.
//	@Description	Fetches and ranks news from all registered sources. Runs every day, including weekends.
//	@Tags			digest
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200		{object}	api.MessageResponse	"Per-source collection results"
//	@Failure		500		{object}	api.MessageResponse	"Failed to collect"
//	@Router			/digest/collect [get]
func (h *Handler) Collect(c *webkit.Context) error {
	ctx := c.Context()

	resp, err := h.runner.Collect(ctx, digest.CollectOptions{})
	if err != nil {
		h.slack.MustSend(ctx, slack.Error("Collect failed", err))
		slog.ErrorContext(ctx, "Collect failed", "err", err)
		return api.Error(c, http.StatusInternalServerError, "Failed to collect")
	}

	hook.Heartbeat(ctx, h.config.BetterStackCollectHeartbeatURL)

	type sourceResult struct {
		Count int     `json:"count"`
		Error *string `json:"error"`
	}
	sources := make(map[string]sourceResult, len(resp.Sources))
	totalItems := 0
	for _, si := range resp.Sources {
		sources[string(si.Source)] = sourceResult{Count: len(si.Items)}
		totalItems += len(si.Items)
	}
	for src, srcErr := range resp.Errors {
		msg := srcErr.Error()
		sources[string(src)] = sourceResult{Error: &msg}
	}
	if len(resp.Errors) > 0 {
		h.slack.MustSend(ctx, sourceErrorsCard(resp.Errors, len(sources), totalItems))
	}

	return api.OK(c, http.StatusOK, map[string]any{"sources": sources}, "Successfully collected sources")
}

// sourceErrorsCard builds the warning Slack card emitted when one or more
// sources fail during a collection run. Each failing source gets its own
// section with the error rendered as a monospaced code block, under a
// summary of how many failed and how many items still landed.
func sourceErrorsCard(errs map[news.Source]error, totalSources, items int) slack.Request {
	const header = "Source errors during collection"

	srcs := make([]string, 0, len(errs))
	for src := range errs {
		srcs = append(srcs, string(src))
	}
	sort.Strings(srcs)

	blocks := make([]slack.Block, 0, 3+len(srcs))
	blocks = append(blocks,
		slackgo.NewHeaderBlock(slackgo.NewTextBlockObject(slackgo.PlainTextType, header, false, false)),
		slackgo.NewContextBlock("", slackgo.NewTextBlockObject(slackgo.MarkdownType,
			fmt.Sprintf("%d of %d sources failed  ·  collection completed with %d items", len(srcs), totalSources, items),
			false, false)),
		slackgo.NewDividerBlock(),
	)
	for _, src := range srcs {
		msg := errs[news.Source(src)].Error()
		blocks = append(blocks, slackgo.NewSectionBlock(
			slackgo.NewTextBlockObject(slackgo.MarkdownType, "*"+src+"*\n```\n"+msg+"\n```", false, false),
			nil, nil,
		))
	}

	fallback := fmt.Sprintf("%s: %d of %d sources failed", header, len(srcs), totalSources)
	return slack.Request{
		Text:        fallback,
		Blocks:      slack.BlockSet{BlockSet: blocks},
		Attachments: []slack.Attachment{{Color: slack.ColorWarn, Fallback: fallback}},
	}
}
