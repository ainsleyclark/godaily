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
	"encoding/xml"
	"os"
	"path/filepath"

	"github.com/ainsleyclark/godaily/pkg/env"
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
		},
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
