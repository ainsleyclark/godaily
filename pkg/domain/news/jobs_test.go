// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package news

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func jobItem(company, title string, score float64) Item {
	var author *Author
	if company != "" {
		author = &Author{Name: company}
	}
	return Item{Title: title, Author: author, Tag: TagJobs, Score: score}
}

func TestJobKey(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		item Item
		want string
	}{
		"Company dot role dot location": {
			item: jobItem("Acme Corp", "Acme Corp · Senior Go Engineer · Remote", 1),
			want: "acme corp\x00senior go engineer",
		},
		"HN pipe convention": {
			item: jobItem("Acme", "Acme | Senior Go Engineer | London | $$$", 1),
			want: "acme\x00senior go engineer",
		},
		"Punctuation and case normalised to match across boards": {
			item: jobItem("Acme, Inc.", "ACME,  INC. · Senior  Go  Engineer", 1),
			want: "acme inc\x00senior go engineer",
		},
		"Non-job tag returns empty": {
			item: Item{Title: "Acme · Go Engineer", Author: &Author{Name: "Acme"}, Tag: TagArticle},
			want: "",
		},
		"Missing company returns empty": {
			item: jobItem("", "Senior Go Engineer", 1),
			want: "",
		},
		"Role equal to company returns empty": {
			item: jobItem("Acme", "Acme", 1),
			want: "",
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, jobKey(test.item))
		})
	}
}

func TestDedupeJobs(t *testing.T) {
	t.Parallel()

	t.Run("Same role across boards collapses to the highest score", func(t *testing.T) {
		t.Parallel()
		// Same company+role on two boards with different URLs/scores.
		in := []SourceItems{
			{Source: SourceRemoteOK, Items: []Item{jobItem("Acme", "Acme · Go Engineer · Remote", 1.0)}},
			{Source: SourceRemotive, Items: []Item{jobItem("Acme", "Acme · Go Engineer · Worldwide", 2.5)}},
		}
		out := DedupeJobs(in)

		// Only the higher-scoring Remotive copy survives; the Remote OK section
		// is dropped entirely as it had a single, now-removed item.
		if assert.Len(t, out, 1) {
			assert.Equal(t, SourceRemotive, out[0].Source)
			if assert.Len(t, out[0].Items, 1) {
				assert.Equal(t, 2.5, out[0].Items[0].Score)
			}
		}
	})

	t.Run("Different roles at the same company are both kept", func(t *testing.T) {
		t.Parallel()
		in := []SourceItems{{Source: SourceRemoteOK, Items: []Item{
			jobItem("Acme", "Acme · Go Engineer · Remote", 1.0),
			jobItem("Acme", "Acme · Platform Engineer · Remote", 1.0),
		}}}
		out := DedupeJobs(in)
		assert.Len(t, out[0].Items, 2)
	})

	t.Run("Thin keys never collapse distinct jobs", func(t *testing.T) {
		t.Parallel()
		// No company → empty key → both pass through untouched.
		in := []SourceItems{{Source: SourceGolangCafe, Items: []Item{
			jobItem("", "Go Engineer", 1.0),
			jobItem("", "Platform Engineer", 1.0),
		}}}
		out := DedupeJobs(in)
		assert.Len(t, out[0].Items, 2)
	})

	t.Run("Non-job items pass through untouched", func(t *testing.T) {
		t.Parallel()
		article := Item{Source: SourceGoBlog, Title: "Go 1.99 is released", Tag: TagArticle, Score: 5}
		in := []SourceItems{{Source: SourceGoBlog, Items: []Item{article, article}}}
		out := DedupeJobs(in)
		assert.Len(t, out[0].Items, 2)
	})

	t.Run("Tie keeps the first (higher-priority) source's listing", func(t *testing.T) {
		t.Parallel()
		in := []SourceItems{
			{Source: SourceGolangCafe, Items: []Item{jobItem("Acme", "Acme · Go Engineer", 1.0)}},
			{Source: SourceRemoteOK, Items: []Item{jobItem("Acme", "Acme · Go Engineer", 1.0)}},
		}
		out := DedupeJobs(in)
		// The first (higher-priority) section keeps its copy; the later one is
		// dropped, so only the Golang.cafe section remains.
		if assert.Len(t, out, 1) {
			assert.Equal(t, SourceGolangCafe, out[0].Source)
		}
	})
}
