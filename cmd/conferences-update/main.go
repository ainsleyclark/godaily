// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// conferences-update reads pkg/data/conferences-watch.yaml (a curated list of
// conference websites) and checks each URL against conferences.yaml. If a URL has
// no entry with a start_date in the current or future year, the conference website
// is fetched and Claude extracts a structured YAML entry which is appended for
// human review.
// Run via: make conferences-update
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"gopkg.in/yaml.v3"
)

const maxHTMLBytes = 16_000

func main() {
	yamlPath := flag.String("yaml", "pkg/data/conferences.yaml", "path to conferences.yaml")
	watchPath := flag.String("watch", "pkg/data/conferences-watch.yaml", "path to conferences-watch.yaml")
	flag.Parse()

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("conferences-update: ANTHROPIC_API_KEY is not set")
	}

	watchURLs, err := loadWatchList(*watchPath)
	if err != nil {
		log.Fatalf("conferences-update: read watch list: %v", err)
	}

	localYAML, err := os.ReadFile(*yamlPath)
	if err != nil {
		log.Fatalf("conferences-update: read %s: %v", *yamlPath, err)
	}
	existing, err := parseExistingConferences(localYAML)
	if err != nil {
		log.Fatalf("conferences-update: parse conferences.yaml: %v", err)
	}

	thisYear := time.Now().UTC().Year()
	var stale []string
	for _, u := range watchURLs {
		if needsEntry(u, existing, thisYear) {
			stale = append(stale, u)
		}
	}

	if len(stale) == 0 {
		log.Println("conferences-update: all watched conferences have a current-year entry")
		return
	}

	log.Printf("conferences-update: %d URL(s) need a %d entry — fetching and extracting\n", len(stale), thisYear)

	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	var added int

	for _, u := range stale {
		entry, err := extractEntry(context.Background(), client, u, thisYear)
		if err != nil {
			log.Printf("conferences-update: skipping %s: %v\n", u, err)
			continue
		}
		if err := appendEntry(*yamlPath, entry); err != nil {
			log.Printf("conferences-update: write failed for %s: %v\n", u, err)
			continue
		}
		log.Printf("conferences-update: added %s\n", u)
		added++
	}

	log.Printf("conferences-update: done — %d new conference(s) added\n", added)
}

func loadWatchList(path string) ([]string, error) {
	root, err := os.OpenRoot(".")
	if err != nil {
		return nil, err
	}
	defer root.Close()
	f, err := root.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	var urls []string
	if err := yaml.Unmarshal(data, &urls); err != nil {
		return nil, err
	}
	return urls, nil
}

// confEntry holds only the fields needed to check for existing entries.
type confEntry struct {
	URL       string `yaml:"url"`
	StartDate string `yaml:"start_date"`
}

func parseExistingConferences(data []byte) ([]confEntry, error) {
	var entries []confEntry
	if err := yaml.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// needsEntry returns true if the watch URL has no entry in conferences.yaml
// with a start_date year >= thisYear.
func needsEntry(url string, existing []confEntry, thisYear int) bool {
	norm := strings.TrimRight(url, "/")
	for _, e := range existing {
		if strings.TrimRight(e.URL, "/") != norm {
			continue
		}
		if len(e.StartDate) >= 4 {
			year, err := strconv.Atoi(e.StartDate[:4])
			if err == nil && year >= thisYear {
				return false
			}
		}
	}
	return true
}

func extractEntry(ctx context.Context, client anthropic.Client, url string, year int) (string, error) {
	html, err := fetchHTML(url)
	if err != nil {
		return "", fmt.Errorf("fetch website: %w", err)
	}

	const system = `You are a data extraction assistant. Extract Go conference information from a conference website's HTML and return exactly one YAML entry. Return only the raw YAML — no markdown fences, no prose, no explanation.`

	user := fmt.Sprintf(`URL: %s
Year: %d

HTML (may be truncated):
%s

Return a single YAML list entry in this exact format (2-space indentation):
- slug: <conference-name-year>
  name: <Full Conference Name Year>
  url: %s
  location: <City, Country>
  start_date: <YYYY-MM-DD>
  end_date: <YYYY-MM-DD>
  description: "<One concise sentence describing the conference.>"
  image_url: ""
  notify_dates:
    - <YYYY-MM-DD>  # announcement (~6 months before start_date)
    - <YYYY-MM-DD>  # reminder (~3 months before start_date)
    - <YYYY-MM-DD>  # alert (~1 week before start_date)

Rules:
- slug must be lowercase, hyphen-separated, end with the year (e.g. gophercon-eu-2026)
- url must be exactly: %s
- Dates must be YYYY-MM-DD; estimate from any partial information on the page if exact dates are unavailable`, url, year, html, url, url)

	resp, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:       anthropic.ModelClaudeSonnet4_6,
		MaxTokens:   int64(512),
		Temperature: anthropic.Float(0),
		System:      []anthropic.TextBlockParam{{Text: system}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(user)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("claude API: %w", err)
	}

	var sb strings.Builder
	for _, b := range resp.Content {
		if b.Type == "text" {
			sb.WriteString(b.Text)
		}
	}

	yamlText := strings.TrimSpace(sb.String())
	yamlText = strings.TrimPrefix(yamlText, "```yaml")
	yamlText = strings.TrimPrefix(yamlText, "```")
	yamlText = strings.TrimSuffix(yamlText, "```")
	yamlText = strings.TrimSpace(yamlText)

	var check []any
	if err := yaml.Unmarshal([]byte(yamlText), &check); err != nil {
		return "", fmt.Errorf("claude returned invalid YAML: %w", err)
	}
	if len(check) == 0 {
		return "", fmt.Errorf("claude returned empty YAML")
	}

	return yamlText, nil
}

func fetchHTML(url string) (string, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; godaily-conferences-bot/1.0)")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxHTMLBytes))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func appendEntry(path, entry string) error {
	root, err := os.OpenRoot(".")
	if err != nil {
		return err
	}
	defer root.Close()
	f, err := root.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "\n%s\n", entry)
	return err
}
