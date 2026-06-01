// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generate

import (
	"encoding/xml"
	"os"
	"path/filepath"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/web/views/pages"
	"github.com/pkg/errors"
)

const sitemapNamespace = "http://www.sitemaps.org/schemas/sitemap/0.9"

type urlSet struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Loc      string `xml:"loc"`
	LastMod  string `xml:"lastmod,omitempty"`
	Priority string `xml:"priority"`
}

// sitemap writes sitemap.xml to outDir containing the homepage and one entry
// per issue at /issues/{slug}/.
func sitemap(w website, outDir string) error {
	home := sitemapURL{Loc: env.AppURL + "/", Priority: "1.0"}
	if len(w.Issues) > 0 {
		home.LastMod = w.Issues[0].SentAt.Format("2006-01-02")
	}

	set := urlSet{
		Xmlns: sitemapNamespace,
		URLs: []sitemapURL{
			home,
			{Loc: env.AppURL + "/issues/", Priority: "0.9"},
			{Loc: env.AppURL + "/browse/", Priority: "0.7"},
		},
	}

	for _, tag := range news.SectionTags {
		set.URLs = append(set.URLs, sitemapURL{
			Loc:      env.AppURL + pages.BrowseTagURL(tag),
			Priority: "0.7",
		})
	}

	for _, issue := range w.Issues {
		set.URLs = append(set.URLs, sitemapURL{
			Loc:      env.AppURL + "/issues/" + issue.Slug + "/",
			LastMod:  issue.SentAt.Format("2006-01-02"),
			Priority: "0.8",
		})
	}

	data, err := xml.MarshalIndent(set, "", "  ")
	if err != nil {
		return errors.Wrap(err, "marshalling sitemap")
	}

	out := append([]byte(xml.Header), data...)
	return errors.Wrap(
		os.WriteFile(filepath.Join(outDir, "sitemap.xml"), out, 0o600),
		"writing sitemap.xml",
	)
}
