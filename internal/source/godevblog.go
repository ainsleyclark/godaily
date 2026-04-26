package source

import (
	"context"
	"encoding/xml"
	"net/http"
	"time"

	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/ainsleydev/webkit/pkg/util/httputil"
	"github.com/pkg/errors"
)

// GoBlog defines the type that implements news.Fetcher.
type GoBlog struct {
	http *http.Client
	url  string
}

var _ news.Fetcher = &GoBlog{}

func init() {
	news.Register(news.SourceGoBlog, func() news.Fetcher { return NewGoBlog() })
}

const goBlogURL = "https://go.dev/blog/feed.atom"

// NewGoBlog creates a Go Dev Blog client.
func NewGoBlog() *GoBlog {
	return &GoBlog{
		http: &http.Client{},
		url:  goBlogURL,
	}
}

// Fetch retrieves all the news items from the Go Dev Blog Atom feed.
func (g GoBlog) Fetch(ctx context.Context) ([]news.Item, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", g.url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "go blog request creation failed")
	}

	resp, err := g.http.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "fetch go blog")
	}
	defer resp.Body.Close()

	if !httputil.Is2xx(resp.StatusCode) {
		return nil, errors.Errorf("unexpected status code from go blog: %d", resp.StatusCode)
	}

	var feed goBlogFeed
	if err = xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return nil, errors.Wrap(err, "parsing response")
	}

	out := make([]news.Item, len(feed.Entries))
	for i, entry := range feed.Entries {
		out[i] = entry.transform()
	}

	return out, nil
}

type (
	goBlogFeed struct {
		XMLName xml.Name      `xml:"http://www.w3.org/2005/Atom feed"`
		Entries []goBlogEntry `xml:"http://www.w3.org/2005/Atom entry"`
	}
	goBlogEntry struct {
		Title     string       `xml:"http://www.w3.org/2005/Atom title"`
		Links     []goBlogLink `xml:"http://www.w3.org/2005/Atom link"`
		Author    goBlogAuthor `xml:"http://www.w3.org/2005/Atom author"`
		Published string       `xml:"http://www.w3.org/2005/Atom published"`
		Summary   string       `xml:"http://www.w3.org/2005/Atom summary"`
	}
	goBlogLink struct {
		Href string `xml:"href,attr"`
		Rel  string `xml:"rel,attr"`
	}
	goBlogAuthor struct {
		Name string `xml:"http://www.w3.org/2005/Atom name"`
	}
)

// url returns the canonical URL of the entry by finding the first link with
// rel="alternate" or an empty rel. Returns an empty string if no such link exists.
func (e goBlogEntry) url() string {
	for _, l := range e.Links {
		if l.Rel == "alternate" || l.Rel == "" {
			return l.Href
		}
	}
	return ""
}

func (e goBlogEntry) transform() news.Item {
	published, _ := time.Parse(time.RFC3339, e.Published)
	return news.Item{
		Source:    news.SourceGoBlog,
		Title:     e.Title,
		URL:       e.url(),
		Author:    e.Author.Name,
		Snippet:   e.Summary,
		Tag:       news.TagArticle,
		Published: published,
	}
}
