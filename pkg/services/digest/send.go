// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/gateway/email"
	"github.com/ainsleyclark/godaily/pkg/store"
)

// SendDigest loads the draft digest for the given date, sends it to all
// active subscribers, then updates the stored issue status to sent.
// The admin preview (digest + synth) is sent separately via SendPreview.
func (s Service) SendDigest(ctx context.Context, date time.Time, force bool) error {
	slug := date.Format("2006-01-02")

	slog.InfoContext(ctx, "Preparing to send digest", "slug", slug)

	issue, sections, err := s.loadDraftDigest(ctx, slug, force)
	if err != nil {
		return err
	}

	subs, err := s.subscribers.ListActive(ctx)
	if err != nil {
		return errors.Wrap(err, "listing active subscribers")
	}

	canonicalURL := env.AppURL + "/issues/" + issue.Slug + "/"

	issueTag := email.Tag{Name: email.TagIssueID, Value: strconv.FormatInt(issue.ID, 10)}

	// Build a personalized request for each active subscriber, skipping any
	// that fail to render. Subscriber errors are non-fatal; the issue is
	// marked sent once all batches are dispatched.
	var batch []*email.SendEmailRequest
	for _, sub := range subs {
		if sub.UnsubscribeToken == "" {
			slog.ErrorContext(ctx, "Skipping subscriber with missing unsubscribe token", "email", sub.Email)
			continue
		}
		// Trailing slash is required: vercel.json sets trailingSlash:true,
		// so /api/unsubscribe responds 308 → /api/unsubscribe/, and Gmail's
		// RFC 8058 one-click POST will not be honoured by a redirect.
		unsubURL := env.AppURL + "/api/unsubscribe/?token=" + sub.UnsubscribeToken
		subRendered, renderErr := renderDigest(digestOptions{
			Day:            date,
			Subject:        issue.Subject,
			Intro:          issue.Summary,
			Sources:        sections,
			UnsubscribeURL: unsubURL,
			CanonicalURL:   canonicalURL,
		})
		if renderErr != nil {
			slog.ErrorContext(ctx, "Failed to render digest for subscriber", "email", sub.Email, "err", renderErr)
			continue
		}
		tags := []email.Tag{issueTag, {Name: email.TagSubscriberID, Value: strconv.FormatInt(sub.ID, 10)}}
		batch = append(batch, buildEmailRequest(sub.Email, subRendered, tags))
	}

	for i := 0; i < len(batch); i += email.BatchSize {
		chunk := batch[i:min(i+email.BatchSize, len(batch))]
		if sendErr := s.email.SendBatch(ctx, chunk); sendErr != nil {
			slog.ErrorContext(ctx, "Failed to send digest batch", "start", i, "end", i+len(chunk), "err", sendErr)
		}
	}

	if _, err = s.issues.UpdateStatus(ctx, issue.ID, digest.IssueStatusSent, time.Now().UTC()); err != nil {
		slog.ErrorContext(ctx, "Failed to update issue status", "err", err)
	}

	return nil
}

// loadDraftDigest finds the issue for slug, validates its status, and returns
// the issue along with its grouped sections. Pass force=true to skip the
// draft-status guard (used by SendDigest --force).
func (s Service) loadDraftDigest(ctx context.Context, slug string, force bool) (digest.Issue, []news.SourceItems, error) {
	issue, err := s.issues.FindBySlug(ctx, slug)
	if errors.Is(err, store.ErrNotFound) {
		return digest.Issue{}, nil, fmt.Errorf("no digest found for %s — run `godaily build` first", slug)
	} else if err != nil {
		return digest.Issue{}, nil, errors.Wrap(err, "loading digest")
	} else if !force && issue.Status != digest.IssueStatusDraft {
		return digest.Issue{}, nil, fmt.Errorf("digest for %s has status %q, expected %q", slug, issue.Status, digest.IssueStatusDraft)
	}

	sections, err := loadSections(ctx, s.items, issue.ID)
	if err != nil {
		return digest.Issue{}, nil, errors.Wrap(err, "loading sections")
	}

	return issue, sections, nil
}

// loadSections fetches stored items for an issue and groups them into
// SourceItems slices sorted by source priority, matching the shape
// produced by Build.
func loadSections(ctx context.Context, repo news.ItemRepository, issueID int64) ([]news.SourceItems, error) {
	items, err := repo.List(ctx, news.ItemListOptions{IssueID: &issueID})
	if err != nil {
		return nil, err
	}

	order := make([]news.Source, 0)
	bySource := make(map[news.Source]*news.SourceItems)
	for _, item := range items {
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

	return sections, nil
}
