// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// conferences-check fetches the Go wiki Conferences page and reports any
// conference website URLs found there that do not appear in conferences.yaml.
// Run via: make conferences-check
package main

import (
	"bufio"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const wikiURL = "https://raw.githubusercontent.com/golang/wiki/master/Conferences.md"

func main() {
	yamlPath := flag.String("yaml", "pkg/data/conferences.yaml", "path to conferences.yaml (relative to repo root)")
	flag.Parse()

	localYAML, err := os.ReadFile(*yamlPath)
	if err != nil {
		log.Fatalf("conferences-check: read %s: %v", *yamlPath, err)
	}

	resp, err := http.Get(wikiURL) //nolint:noctx
	if err != nil {
		log.Fatalf("conferences-check: fetch wiki: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("conferences-check: wiki returned %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("conferences-check: read body: %v", err)
	}

	wikiURLs := extractURLs(string(body))
	localURLs := localConferenceURLs(localYAML)

	var missing []string
	for _, u := range wikiURLs {
		if !localURLs[u] {
			missing = append(missing, u)
		}
	}

	if len(missing) == 0 {
		log.Println("conferences-check: all wiki conferences are present in conferences.yaml")
		return
	}

	log.Printf("conferences-check: %d URL(s) found in the Go wiki but not in conferences.yaml:\n", len(missing))
	for _, u := range missing {
		log.Printf("  - %s\n", u)
	}
}

var mdLinkRe = regexp.MustCompile(`\[.*?\]\((https?://[^\)]+)\)`)

// extractURLs parses Markdown and returns unique HTTP(S) URLs found in links.
func extractURLs(md string) []string {
	seen := map[string]bool{}
	var out []string
	scanner := bufio.NewScanner(strings.NewReader(md))
	for scanner.Scan() {
		line := scanner.Text()
		for _, m := range mdLinkRe.FindAllStringSubmatch(line, -1) {
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

// localConferenceURLs returns a set of URLs from conferences.yaml (normalised
// to bare URL without fragments).
func localConferenceURLs(data []byte) map[string]bool {
	var entries []confEntry
	if err := yaml.Unmarshal(data, &entries); err != nil {
		log.Fatalf("conferences-check: parse conferences.yaml: %v", err)
	}
	out := make(map[string]bool, len(entries))
	for _, e := range entries {
		u := strings.TrimRight(e.URL, "/")
		out[u] = true
		out[u+"/"] = true
	}
	return out
}
