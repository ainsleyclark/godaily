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

// conferences-update fetches the Go wiki Conferences page, identifies any
// conference URLs not already in conferences.yaml, fetches each conference
// website, and uses the Claude API to extract a structured YAML entry for
// each one. New entries are appended to conferences.yaml for human review.
// Run via: make conferences-update
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"gopkg.in/yaml.v3"
)

const (
	wikiURL      = "https://raw.githubusercontent.com/golang/wiki/master/Conferences.md"
	maxHTMLBytes = 16_000
)

var mdLinkRe = regexp.MustCompile(`\[.*?\]\((https?://[^\)]+)\)`)

func main() {
	yamlPath := flag.String("yaml", "pkg/source/conferences.yaml", "path to conferences.yaml (relative to repo root)")
	flag.Parse()

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("conferences-update: ANTHROPIC_API_KEY is not set")
	}

	localYAML, err := os.ReadFile(*yamlPath)
	if err != nil {
		log.Fatalf("conferences-update: read %s: %v", *yamlPath, err)
	}

	wikiURLs := fetchWikiURLs()
	localURLs := localConferenceURLs(localYAML)

	var missing []string
	for _, u := range wikiURLs {
		if !localURLs[u] {
			missing = append(missing, u)
		}
	}

	if len(missing) == 0 {
		log.Println("conferences-update: all wiki conferences are already in conferences.yaml")
		return
	}

	log.Printf("conferences-update: %d URL(s) not in conferences.yaml — fetching and extracting\n", len(missing))

	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	var added int

	for _, u := range missing {
		entry, err := extractEntry(context.Background(), client, u)
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

func fetchWikiURLs() []string {
	resp, err := http.Get(wikiURL) //nolint:noctx
	if err != nil {
		log.Fatalf("conferences-update: fetch wiki: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("conferences-update: wiki returned %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("conferences-update: read wiki body: %v", err)
	}
	return extractURLs(string(body))
}

func extractURLs(md string) []string {
	seen := map[string]bool{}
	var out []string
	scanner := bufio.NewScanner(strings.NewReader(md))
	for scanner.Scan() {
		for _, m := range mdLinkRe.FindAllStringSubmatch(scanner.Text(), -1) {
			u := strings.TrimSpace(m[1])
			if !seen[u] {
				seen[u] = true
				out = append(out, u)
			}
		}
	}
	return out
}

type confEntry struct {
	URL string `yaml:"url"`
}

func localConferenceURLs(data []byte) map[string]bool {
	var entries []confEntry
	if err := yaml.Unmarshal(data, &entries); err != nil {
		log.Fatalf("conferences-update: parse conferences.yaml: %v", err)
	}
	out := make(map[string]bool, len(entries))
	for _, e := range entries {
		u := strings.TrimRight(e.URL, "/")
		out[u] = true
		out[u+"/"] = true
	}
	return out
}

func extractEntry(ctx context.Context, client anthropic.Client, url string) (string, error) {
	html, err := fetchHTML(url)
	if err != nil {
		return "", fmt.Errorf("fetch website: %w", err)
	}

	year := time.Now().UTC().Year()

	const system = `You are a data extraction assistant. Extract Go conference information from a conference website's HTML and return exactly one YAML entry. Return only the raw YAML — no markdown fences, no prose, no explanation.`

	user := fmt.Sprintf(`URL: %s
Year hint: %d

HTML (may be truncated):
%s

Return a single YAML list entry in this exact format (preserve indentation, use 2 spaces):
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
- slug must be unique, lowercase, hyphen-separated, include the year (e.g. gophercon-eu-2026)
- url must be exactly: %s
- If exact dates are unavailable, estimate from any partial date information on the page
- Dates must be YYYY-MM-DD`, url, year, html, url, url)

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
	// Strip markdown fences if Claude included them despite instructions.
	yamlText = strings.TrimPrefix(yamlText, "```yaml")
	yamlText = strings.TrimPrefix(yamlText, "```")
	yamlText = strings.TrimSuffix(yamlText, "```")
	yamlText = strings.TrimSpace(yamlText)

	// Validate it parses as a YAML list.
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
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "\n%s\n", entry)
	return err
}
