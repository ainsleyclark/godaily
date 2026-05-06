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

// Package generate renders all godaily pages as static HTML files for
// deployment to Vercel's CDN.
package generate

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/a-h/templ"
	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/ainsleyclark/godaily/web/views/pages"
	"github.com/pkg/errors"
)

// Site renders all sent issues and the homepage to outDir, then copies
// compiled frontend assets from assetsDir into outDir/assets.
func Site(ctx context.Context, repo news.IssueRepository, outDir, assetsDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return errors.Wrap(err, "creating output directory")
	}

	allIssues, err := repo.List(ctx)
	if err != nil {
		return errors.Wrap(err, "listing issues")
	}

	slog.InfoContext(ctx, "Generating static site", "issues", len(allIssues), "out", outDir)

	latest, err := repo.Latest(ctx, 1)
	if err != nil {
		return errors.Wrap(err, "fetching latest issue")
	}

	var latestIssue news.Issue
	if len(latest) > 0 {
		latestIssue = latest[0]
	}

	homeData := pages.HomeData{
		LatestIssue: latestIssue,
		SampleIssue: latestIssue,
	}
	if err := renderPage(ctx, filepath.Join(outDir, "index.html"), pages.Home(homeData)); err != nil {
		return errors.Wrap(err, "rendering homepage")
	}

	for _, issue := range allIssues {
		full, err := repo.Find(ctx, issue.ID)
		if err != nil {
			return fmt.Errorf("fetching issue %d: %w", issue.ID, err)
		}
		dir := filepath.Join(outDir, "digest", issue.Slug)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating digest directory for %s: %w", issue.Slug, err)
		}
		if err := renderPage(ctx, filepath.Join(dir, "index.html"), pages.Digest(full)); err != nil {
			return fmt.Errorf("rendering digest %s: %w", issue.Slug, err)
		}
		slog.InfoContext(ctx, "Rendered digest", "slug", issue.Slug)
	}

	if err := copyDir(assetsDir, filepath.Join(outDir, "assets")); err != nil {
		return errors.Wrap(err, "copying assets")
	}

	return nil
}

func renderPage(ctx context.Context, path string, component templ.Component) error {
	var buf bytes.Buffer
	if err := component.Render(ctx, &buf); err != nil {
		return errors.Wrap(err, "rendering component")
	}
	return errors.Wrap(os.WriteFile(path, buf.Bytes(), 0o644), "writing file")
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(src, path)
		if relErr != nil {
			return relErr
		}
		dstPath := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(dstPath, 0o755)
		}
		return copyFile(path, dstPath)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return errors.Wrap(err, "opening source file")
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return errors.Wrap(err, "creating destination directory")
	}

	out, err := os.Create(dst)
	if err != nil {
		return errors.Wrap(err, "creating destination file")
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return errors.Wrap(err, "copying file contents")
}
