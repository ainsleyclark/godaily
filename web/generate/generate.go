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
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ainsleyclark/godaily/pkg/news"
	"github.com/ainsleyclark/godaily/web/views/pages"
	"github.com/pkg/errors"
)

// website holds all data required to generate the static site.
type website struct {
	Issues       []news.Issue
	LatestIssue  news.Issue
	RecentIssues []news.Issue
}

// Site renders all sent issues and the homepage to outDir, generates
// sitemap.xml and rss.xml, copies static files from staticDir, then
// copies compiled frontend assets from assetsDir into outDir/assets.
func Site(ctx context.Context, repo news.IssueRepository, subscriberCount int64, outDir, staticDir, assetsDir string) error {
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		return errors.Wrap(err, "creating output directory")
	}

	allIssues, err := repo.List(ctx)
	if err != nil {
		return errors.Wrap(err, "listing issues")
	}

	slog.InfoContext(ctx, "Generating static site", "issues", len(allIssues), "out", outDir)

	recent, err := repo.Latest(ctx, 4)
	if err != nil {
		return errors.Wrap(err, "fetching latest issue")
	}

	w := website{Issues: allIssues, RecentIssues: recent}
	if len(recent) > 0 {
		w.LatestIssue = recent[0]
	}

	if err := renderPages(ctx, repo, w, subscriberCount, outDir); err != nil {
		return errors.Wrap(err, "rendering pages")
	}

	if err := renderPage(ctx, filepath.Join(outDir, "404.html"), pages.Error(http.StatusNotFound)); err != nil {
		return errors.Wrap(err, "rendering 404 page")
	}

	if err := sitemap(w, outDir); err != nil {
		return errors.Wrap(err, "generating sitemap")
	}

	if err := rss(w, outDir); err != nil {
		return errors.Wrap(err, "generating RSS feed")
	}

	if err := copyDir(staticDir, outDir); err != nil {
		return errors.Wrap(err, "copying static files")
	}

	if err := copyDir(assetsDir, filepath.Join(outDir, "assets")); err != nil {
		return errors.Wrap(err, "copying assets")
	}

	return nil
}

// copyDir copies all files from src into dst using os.Root to scope both
// directories, preventing directory traversal (gosec G304).
func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0o750); err != nil {
		return errors.Wrap(err, "creating destination root")
	}

	srcRoot, err := os.OpenRoot(src)
	if err != nil {
		return errors.Wrap(err, "opening source root")
	}
	defer srcRoot.Close()

	dstRoot, err := os.OpenRoot(dst)
	if err != nil {
		return errors.Wrap(err, "opening destination root")
	}
	defer dstRoot.Close()

	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(src, path)
		if relErr != nil {
			return relErr
		}
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			return dstRoot.MkdirAll(rel, 0o750)
		}
		return copyFile(srcRoot, dstRoot, rel)
	})
}

func copyFile(src, dst *os.Root, rel string) error {
	in, err := src.Open(rel)
	if err != nil {
		return errors.Wrap(err, "opening source file")
	}
	defer in.Close()

	out, err := dst.Create(rel)
	if err != nil {
		return errors.Wrap(err, "creating destination file")
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return errors.Wrap(err, "copying file contents")
}
