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
	"time"

	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/pkg/errors"
)

type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title         string    `xml:"title"`
	Link          string    `xml:"link"`
	Description   string    `xml:"description"`
	Language      string    `xml:"language"`
	LastBuildDate string    `xml:"lastBuildDate,omitempty"`
	Items         []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description,omitempty"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

// rss writes rss.xml to outDir as an RSS 2.0 feed with one item per issue.
func rss(w website, outDir string) error {
	channel := rssChannel{
		Title:       "GoDaily",
		Link:        env.AppURL,
		Description: "A daily digest of the best Go news, articles, and community content.",
		Language:    "en",
	}

	if len(w.Issues) > 0 {
		channel.LastBuildDate = w.Issues[0].SentAt.UTC().Format(time.RFC1123Z)
	}

	for _, issue := range w.Issues {
		link := env.AppURL + "/issues/" + issue.Slug + "/"
		channel.Items = append(channel.Items, rssItem{
			Title:       issue.Subject,
			Link:        link,
			Description: issue.Summary,
			PubDate:     issue.SentAt.UTC().Format(time.RFC1123Z),
			GUID:        link,
		})
	}

	feed := rssFeed{
		Version: "2.0",
		Channel: channel,
	}

	data, err := xml.MarshalIndent(feed, "", "  ")
	if err != nil {
		return errors.Wrap(err, "marshalling RSS feed")
	}

	out := append([]byte(xml.Header), data...)
	return errors.Wrap(
		os.WriteFile(filepath.Join(outDir, "rss.xml"), out, 0o600),
		"writing rss.xml",
	)
}
