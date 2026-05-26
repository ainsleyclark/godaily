// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generate

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/store"
	"github.com/ainsleyclark/godaily/web/views/pages"
	"github.com/pkg/errors"
)

// website holds all data required to generate the static site.
type website struct {
	Issues       []digest.Issue
	LatestIssue  digest.Issue
	RecentIssues []digest.Issue
}

// Site renders all sent issues and the homepage to outDir, generates
// sitemap.xml and rss.xml, copies static files from staticDir, then
// copies compiled frontend assets from assetsDir into outDir/assets.
func Site(ctx context.Context, repo digest.IssueRepository, subscriberCount int64, outDir, staticDir, assetsDir string) error {
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		return errors.Wrap(err, "creating output directory")
	}

	allIssues, err := repo.List(ctx, store.ListOptions{})
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
