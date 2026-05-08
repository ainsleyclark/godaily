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

package generate

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/a-h/templ"
	"github.com/ainsleyclark/godaily/pkg/news"
	"github.com/ainsleyclark/godaily/web/views/pages"
	"github.com/pkg/errors"
)

// renderPages writes the homepage, thank-you page, and all individual digest
// pages to outDir. It calls repo.Find for each issue to load its full item list.
func renderPages(ctx context.Context, repo news.IssueRepository, w website, outDir string) error {
	homeData := pages.HomeData{
		LatestIssue: w.LatestIssue,
		SampleIssue: w.LatestIssue,
	}
	if err := renderPage(ctx, filepath.Join(outDir, "index.html"), pages.Home(homeData)); err != nil {
		return errors.Wrap(err, "rendering homepage")
	}

	if err := renderPageInDir(ctx, filepath.Join(outDir, "thank-you"), pages.ThankYou(w.LatestIssue)); err != nil {
		return errors.Wrap(err, "rendering thank-you page")
	}

	if err := renderPageInDir(ctx, filepath.Join(outDir, "unsubscribed"), pages.Unsubscribed()); err != nil {
		return errors.Wrap(err, "rendering unsubscribed page")
	}

	for _, issue := range w.Issues {
		full, err := repo.Find(ctx, issue.ID)
		if err != nil {
			return fmt.Errorf("fetching issue %d: %w", issue.ID, err)
		}
		if err := renderPageInDir(ctx, filepath.Join(outDir, "digest", issue.Slug), pages.Digest(full)); err != nil {
			return fmt.Errorf("rendering digest %s: %w", issue.Slug, err)
		}
		slog.InfoContext(ctx, "Rendered digest", "slug", issue.Slug)
	}

	return nil
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
