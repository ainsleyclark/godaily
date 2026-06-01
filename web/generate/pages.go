// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generate

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"

	"github.com/a-h/templ"
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/web/handlers"
	"github.com/ainsleyclark/godaily/web/og"
	"github.com/ainsleyclark/godaily/web/views/pages"
	"github.com/pkg/errors"
)

// renderPages writes the homepage, thank-you page, issues archive, and all
// individual issue pages to outDir. It calls repo.Find for each issue to load its full item list.
func renderPages(ctx context.Context, repo digest.IssueRepository, items news.ItemRepository, w website, subscriberCount int64, outDir string) error {
	gen, err := og.New()
	if err != nil {
		return errors.Wrap(err, "creating OG image generator")
	}

	homeData := pages.HomeData{
		LatestIssue:     w.LatestIssue,
		SampleIssue:     w.LatestIssue,
		RecentIssues:    w.RecentIssues,
		SubscriberCount: subscriberCount,
	}
	if err := renderPage(ctx, filepath.Join(outDir, "index.html"), pages.Home(homeData)); err != nil {
		return errors.Wrap(err, "rendering homepage")
	}
	if err := writeOGImage(outDir, "home.png", func() ([]byte, error) { return gen.Home() }); err != nil {
		return errors.Wrap(err, "generating home OG image")
	}

	if err := renderPageInDir(ctx, filepath.Join(outDir, "thank-you"), pages.ThankYou("")); err != nil {
		return errors.Wrap(err, "rendering thank-you page")
	}

	if err := renderPageInDir(ctx, filepath.Join(outDir, "confirmed"), pages.Confirmed(w.LatestIssue)); err != nil {
		return errors.Wrap(err, "rendering confirmed page")
	}

	if err := renderPageInDir(ctx, filepath.Join(outDir, "unsubscribed"), pages.Unsubscribed()); err != nil {
		return errors.Wrap(err, "rendering unsubscribed page")
	}

	if err := renderPageInDir(ctx, filepath.Join(outDir, "privacy"), pages.Privacy()); err != nil {
		return errors.Wrap(err, "rendering privacy page")
	}

	// The browse page renders its initial (unfiltered) state statically; the
	// client re-fetches filtered fragments from /api/browse on interaction.
	browseProps, err := handlers.BuildBrowseProps(ctx, repo, items, url.Values{})
	if err != nil {
		return errors.Wrap(err, "building browse props")
	}
	if err := renderPageInDir(ctx, filepath.Join(outDir, "browse"), pages.Browse(browseProps)); err != nil {
		return errors.Wrap(err, "rendering browse page")
	}
	for _, tag := range news.SectionTags {
		tagProps, tagErr := handlers.BuildBrowseProps(ctx, repo, items, url.Values{"tab": []string{string(tag)}})
		if tagErr != nil {
			return fmt.Errorf("building browse props for tag %s: %w", tag, tagErr)
		}
		if tagErr = renderPageInDir(ctx, filepath.Join(outDir, "browse", string(tag)), pages.Browse(tagProps)); tagErr != nil {
			return fmt.Errorf("rendering browse/%s page: %w", tag, tagErr)
		}
	}

	fullIssues := make([]digest.Issue, 0, len(w.Issues))
	for _, issue := range w.Issues {
		full, err := repo.Find(ctx, issue.ID)
		if err != nil {
			return fmt.Errorf("fetching issue %d: %w", issue.ID, err)
		}
		fullIssues = append(fullIssues, full)
	}

	if err := renderPageInDir(ctx, filepath.Join(outDir, "issues"), pages.IssuesArchive(fullIssues)); err != nil {
		return errors.Wrap(err, "rendering issues archive page")
	}

	for _, full := range fullIssues {
		if err := renderPageInDir(ctx, filepath.Join(outDir, "issues", full.Slug), pages.Digest(full)); err != nil {
			return fmt.Errorf("rendering issue %s: %w", full.Slug, err)
		}
		issueCopy := full
		if err := writeOGImage(outDir, filepath.Join("issues", full.Slug+".png"), func() ([]byte, error) {
			return gen.Issue(issueCopy)
		}); err != nil {
			return fmt.Errorf("generating OG image for issue %s: %w", full.Slug, err)
		}
		slog.InfoContext(ctx, "Rendered issue", "slug", full.Slug)
	}

	return nil
}

// writeOGImage generates a PNG via fn and writes it to outDir/og/name.
func writeOGImage(outDir, name string, fn func() ([]byte, error)) error {
	png, err := fn()
	if err != nil {
		return errors.Wrap(err, "generating image")
	}
	dest := filepath.Join(outDir, "og", name)
	if err := os.MkdirAll(filepath.Dir(dest), 0o750); err != nil {
		return errors.Wrap(err, "creating OG directory")
	}
	return errors.Wrap(os.WriteFile(dest, png, 0o600), "writing OG image")
}

func renderPageInDir(ctx context.Context, dir string, component templ.Component) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return errors.Wrap(err, "creating directory")
	}
	return renderPage(ctx, filepath.Join(dir, "index.html"), component)
}

func renderPage(ctx context.Context, path string, component templ.Component) error {
	var buf bytes.Buffer
	if err := component.Render(ctx, &buf); err != nil {
		return errors.Wrap(err, "rendering component")
	}
	return errors.Wrap(os.WriteFile(path, buf.Bytes(), 0o600), "writing file")
}
