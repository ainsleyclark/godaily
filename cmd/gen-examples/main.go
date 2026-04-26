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

// gen-examples fetches live data from every registered source and writes the
// results to examples/<source>.json at the project root. Run via:
//
//	go generate ./internal/source/...
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"github.com/ainsleyclark/godaily/internal/news"
	_ "github.com/ainsleyclark/godaily/internal/source"
)

func main() {
	outDir := filepath.Join("..", "..", "internal", "examples")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatalf("create examples dir: %v", err)
	}

	ctx := context.Background()

	for _, s := range news.Sources {
		fetcher, err := news.Get(s)
		if err != nil {
			log.Printf("skipping %s: %v", s, err)
			continue
		}

		items, err := fetcher.Fetch(ctx)
		if err != nil {
			log.Printf("fetch %s: %v", s, err)
			continue
		}

		data, err := json.MarshalIndent(items, "", "\t")
		if err != nil {
			log.Printf("marshal %s: %v", s, err)
			continue
		}

		path := filepath.Join(outDir, string(s)+".json")
		if err := os.WriteFile(path, data, os.ModePerm); err != nil {
			log.Printf("write %s: %v", path, err)
			continue
		}

		log.Printf("wrote %s (%d items)", path, len(items))
	}
}
