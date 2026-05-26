// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
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
