// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"context"
	"log/slog"
	"sort"
	"time"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
	"github.com/ainsleyclark/godaily/pkg/services/digest/internal/slackdata"
	"github.com/ainsleyclark/godaily/pkg/services/digest/prompts"
	"github.com/ainsleyclark/godaily/pkg/store"
)

// Build loads collected items for the appropriate date window, ranks and
// deduplicates them, runs AI synthesis, and persists a draft Issue with the
// items associated. If a draft already exists for the date's slug it is deleted
// and rebuilt. On a successful build it also fires the owner preview as a
// best-effort side effect, so the build cron doubles as the preview cron.
// On any failure a Slack notification is sent.
func (s Service) Build(ctx context.Context, date time.Time) error {
	today := date.UTC().Truncate(24 * time.Hour)
	start, end := digest.BuildWindow(today)
	slug := today.Format("2006-01-02")

	slog.InfoContext(ctx, "Building digest", "slug", slug, "start", start.Format("2006-01-02"), "end", end.Format("2006-01-02"))

	items, err := s.items.List(ctx, news.ItemListOptions{From: &start, To: &end})
	if err != nil {
		return s.buildErr(ctx, errors.Wrap(err, "listing items"))
	}

	if len(items) == 0 {
		slog.WarnContext(ctx, "No items found for build window, skipping", "slug", slug)
		return nil
	}

	slog.InfoContext(ctx, "Found news items", "slug", slug, "items count", len(items))

	sections := groupIntoSections(items)

	subject, summary := s.synthesiseDigestMeta(ctx, today, sections)

	existing, lookupErr := s.issues.FindBySlug(ctx, slug)
	switch {
	case lookupErr == nil:
		slog.InfoContext(ctx, "Replacing existing draft", "slug", slug)
		if err = s.items.DeleteByIssue(ctx, existing.ID); err != nil {
			return s.buildErr(ctx, errors.Wrap(err, "deleting existing items"))
		}
		if _, err = s.issues.Delete(ctx, existing.ID); err != nil {
			return s.buildErr(ctx, errors.Wrap(err, "deleting existing issue"))
		}
	case !errors.Is(lookupErr, store.ErrNotFound):
		return s.buildErr(ctx, errors.Wrap(lookupErr, "checking existing issue"))
	}

	issue := digest.Issue{
		Slug:    slug,
		Subject: subject,
		Summary: summary,
		Status:  digest.IssueStatusDraft,
		SentAt:  today,
	}

	created, err := s.issues.Create(ctx, issue)
	if err != nil {
		return s.buildErr(ctx, errors.Wrap(err, "creating issue"))
	}

	// SelectForDigest is the single definition of what makes up the digest:
	// it groups the window's items into canonical sections, scores and caps
	// each, and returns the survivors in render order. We link exactly that set
	// (position = index), so the persisted rows alone determine what ships —
	// the email and web are pure renderers ordering by position, and the
	// runner-ups stay raw (issue_id NULL) as the issue's candidate pool.
	selected := news.SelectForDigest(flattenSections(sections))
	for i, item := range selected {
		id := created.ID
		if _, err = s.items.Create(ctx, &id, i+1, item); err != nil {
			return s.buildErr(ctx, errors.Wrap(err, "associating item"))
		}
	}
	itemCount := len(selected)

	slog.InfoContext(ctx, "Built draft issue", "slug", slug, "items", itemCount)

	// Preview is best-effort: a failed owner email must not fail the build,
	// since the draft is already persisted and SendDigest can still run.
	if previewErr := s.SendPreview(ctx, today); previewErr != nil {
		slog.WarnContext(ctx, "Sending preview after build failed", "err", previewErr)
		if s.slack != nil {
			s.slack.MustSend(ctx, slack.Error("Send preview after build failed", previewErr))
		}
	}

	// Draft every social post (featured + rotation, where applicable)
	// as a best-effort side effect. A failed AI draft must not fail the
	// build — the email digest still ships at 08:00 and the 11:00
	// publish cron will simply find no drafts to publish. social is
	// optional and may be unset in tests.
	if s.social != nil {
		if _, draftErr := s.social.DraftAll(ctx, social.PostOptions{Date: today}); draftErr != nil {
			slog.WarnContext(ctx, "Drafting social posts after build failed", "err", draftErr)
			if s.slack != nil {
				s.slack.MustSend(ctx, slack.Error("Draft social posts after build failed", draftErr))
			}
		}
		s.sendbuildSummary(ctx, created, itemCount)
	}

	return nil
}

// sendBuildSummary fires the rich Slack card at the end of a successful
// build: digest meta + every draft awaiting publish with an "Edit"
// deep-link into the dashboard. Best-effort — a Slack failure must not
// fail the build.
func (s Service) sendbuildSummary(ctx context.Context, issue digest.Issue, itemCount int) {
	if s.slack == nil || s.posts == nil {
		return
	}

	draftStatus := social.PostStatusDraft
	drafts, err := s.posts.List(ctx, social.PostListOptions{Status: &draftStatus})
	if err != nil {
		slog.WarnContext(ctx, "Loading drafts for build summary failed", "err", err)
		return
	}

	summary := slackdata.BuildSummary(slackdata.Summary{
		IssueDate: issue.Slug,
		IssueID:   issue.ID,
		IssueSlug: issue.Slug,
		Subject:   issue.Subject,
		Intro:     issue.Summary,
		ItemCount: itemCount,
		Drafts:    toSummaryDrafts(drafts),
	})
	s.slack.MustSend(ctx, summary)
}

func toSummaryDrafts(rows []social.Post) []slackdata.Draft {
	out := make([]slackdata.Draft, 0, len(rows))
	for _, r := range rows {
		out = append(out, slackdata.Draft{
			ID:       r.ID,
			Kind:     string(r.Kind),
			Platform: r.Platform,
			Text:     r.Text,
		})
	}
	return out
}

// flattenSections collapses the per-source sections back into a single item
// slice, stamping each item with its source. The slice is the deduplicated set
// of window items, ready for news.SelectForDigest to rank, cap and order.
func flattenSections(sections []news.SourceItems) []news.Item {
	var flat []news.Item
	for _, section := range sections {
		for _, item := range section.Items {
			item.Source = section.Source
			flat = append(flat, item)
		}
	}
	return flat
}

// groupIntoSections groups a flat item list into SourceItems slices,
// preserving per-source score ordering and deduplicating by (URL, tag).
// Using (URL, tag) rather than URL alone allows both a TagEvent announcement
// and a future TagEventRecap to appear in the same digest for the same URL.
func groupIntoSections(items []news.Item) []news.SourceItems {
	seen := make(map[string]struct{})
	order := make([]news.Source, 0)
	bySource := make(map[news.Source]*news.SourceItems)

	for _, item := range items {
		key := item.URL + "\x00" + string(item.Tag)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}

		if _, ok := bySource[item.Source]; !ok {
			bySource[item.Source] = &news.SourceItems{Source: item.Source}
			order = append(order, item.Source)
		}
		bySource[item.Source].Items = append(bySource[item.Source].Items, item)
	}

	sections := make([]news.SourceItems, 0, len(bySource))
	for _, src := range order {
		sections = append(sections, *bySource[src])
	}

	sort.SliceStable(sections, func(i, j int) bool {
		return sections[i].Source.Priority() > sections[j].Source.Priority()
	})

	return sections
}

func (s Service) buildErr(ctx context.Context, err error) error {
	if s.slack != nil {
		s.slack.MustSend(ctx, slack.Error("Build failed", err))
	}
	return err
}

// synthesiseDigestMeta calls the prompter to generate the email subject title
// and intro paragraph. On failure it logs a warning and returns static fallbacks
// so a missing API key never blocks delivery.
func (s Service) synthesiseDigestMeta(ctx context.Context, day time.Time, sections []news.SourceItems) (subject, summary string) {
	subject = "GoDaily - " + day.Format("January 2, 2006")
	if s.prompter == nil {
		return subject, ""
	}

	meta, err := prompts.Synthesise(ctx, s.prompter, day, sections)
	if err != nil {
		slog.WarnContext(ctx, "Synth digest meta failed, using static subject", "err", err)
		if s.slack != nil {
			s.slack.MustSend(ctx, slack.Error("AI synthesis failed", err))
		}
		return subject, ""
	}

	return meta.Title, meta.Intro
}
