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

package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ainsleyclark/godaily/internal/news"
)

func prettyJSON(v any) []byte {
	b, err := json.MarshalIndent(v, "", "\t")
	exit(context.Background(), err)
	return b
}

// parseSources validates a slice of raw source name strings against the
// registered sources list and returns the typed slice.
func parseSources(raw []string) ([]news.Source, error) {
	known := make(map[news.Source]struct{}, len(news.Sources))
	for _, s := range news.Sources {
		known[s] = struct{}{}
	}
	sources := make([]news.Source, 0, len(raw))
	for _, name := range raw {
		s := news.Source(name)
		if _, ok := known[s]; !ok {
			return nil, fmt.Errorf("unknown source %q (run `godaily sources` for the list)", name)
		}
		sources = append(sources, s)
	}
	return sources, nil
}
