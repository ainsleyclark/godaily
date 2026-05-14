package layouts

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/news"
)

const orgLogoURL = "https://godaily.dev/assets/favicon/favicon-96x96.png"

// OrgSchema is the site-wide Organization JSON-LD rendered on every page.
const OrgSchema = `{"@context":"https://schema.org","@type":"Organization","name":"GoDaily","url":"https://godaily.dev/","description":"A free daily Go newsletter delivering the best Go news and articles from across the community.","logo":{"@type":"ImageObject","url":"https://godaily.dev/assets/favicon/favicon-96x96.png","width":96,"height":96}}`

// WebSiteSchema returns the WebSite JSON-LD for the homepage.
func WebSiteSchema() string {
	schema := map[string]any{
		"@context": "https://schema.org",
		"@type":    "WebSite",
		"name":     "GoDaily",
		"url":      env.AppURL + "/",
		"description": "The daily Go newsletter — best stories from the Go community, ranked and delivered before standup.",
		"potentialAction": map[string]any{
			"@type": "SearchAction",
			"target": map[string]any{
				"@type":       "EntryPoint",
				"urlTemplate": env.AppURL + "/issues/?q={search_term_string}",
			},
			"query-input": "required name=search_term_string",
		},
	}
	return marshalSchema(schema)
}

// IssueSchemas returns a JSON-LD array containing NewsArticle and BreadcrumbList
// schemas for a single digest page.
func IssueSchemas(issue news.Issue) string {
	issueURL := fmt.Sprintf("%s/issues/%s/", env.AppURL, issue.Slug)
	ogImage := fmt.Sprintf("%s/og/issues/%s.png", env.AppURL, issue.Slug)
	publisher := map[string]any{
		"@type": "Organization",
		"name":  "GoDaily",
		"url":   env.AppURL + "/",
		"logo": map[string]any{
			"@type": "ImageObject",
			"url":   orgLogoURL,
		},
	}

	article := map[string]any{
		"@context":      "https://schema.org",
		"@type":         "NewsArticle",
		"headline":      issue.Subject,
		"description":   issue.Summary,
		"url":           issueURL,
		"datePublished": issue.SentAt.UTC().Format(time.RFC3339),
		"dateModified":  issue.SentAt.UTC().Format(time.RFC3339),
		"image":         ogImage,
		"publisher":     publisher,
		"author":        publisher,
	}

	breadcrumb := map[string]any{
		"@context": "https://schema.org",
		"@type":    "BreadcrumbList",
		"itemListElement": []map[string]any{
			{"@type": "ListItem", "position": 1, "name": "Home", "item": env.AppURL + "/"},
			{"@type": "ListItem", "position": 2, "name": "Issues", "item": env.AppURL + "/issues/"},
			{"@type": "ListItem", "position": 3, "name": fmt.Sprintf("Issue #%d", issue.ID), "item": issueURL},
		},
	}

	schemas := []any{article, breadcrumb}
	return marshalSchema(schemas)
}

// ArchiveSchemas returns a JSON-LD array containing CollectionPage and BreadcrumbList
// schemas for the issues archive page.
func ArchiveSchemas(issues []news.Issue) string {
	archiveURL := env.AppURL + "/issues/"

	collection := map[string]any{
		"@context":    "https://schema.org",
		"@type":       "CollectionPage",
		"name":        "GoDaily — Go Newsletter Archive",
		"description": "Browse every issue of GoDaily, the daily Go newsletter.",
		"url":         archiveURL,
	}

	breadcrumb := map[string]any{
		"@context": "https://schema.org",
		"@type":    "BreadcrumbList",
		"itemListElement": []map[string]any{
			{"@type": "ListItem", "position": 1, "name": "Home", "item": env.AppURL + "/"},
			{"@type": "ListItem", "position": 2, "name": "Issues", "item": archiveURL},
		},
	}

	schemas := []any{collection, breadcrumb}
	return marshalSchema(schemas)
}

func marshalSchema(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
