// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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
		Language:    "en-gb",
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
