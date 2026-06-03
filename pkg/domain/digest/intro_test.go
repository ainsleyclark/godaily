// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIntroParagraphs(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		summary string
		want    []string
	}{
		"Empty": {
			summary: "",
			want:    nil,
		},
		"Whitespace only": {
			summary: "  \n\t\n ",
			want:    nil,
		},
		"Single paragraph": {
			summary: "One tight thought about the release.",
			want:    []string{"One tight thought about the release."},
		},
		"Blank line separates subjects": {
			summary: "First subject.\n\nSecond subject.",
			want:    []string{"First subject.", "Second subject."},
		},
		"Single break is treated like a paragraph": {
			summary: "First subject.\nSecond subject.",
			want:    []string{"First subject.", "Second subject."},
		},
		"Collapses extra blank lines and trims": {
			summary: "  First.  \n\n\n   Second.   \n",
			want:    []string{"First.", "Second."},
		},
		"Normalises carriage returns": {
			summary: "First.\r\n\r\nSecond.",
			want:    []string{"First.", "Second."},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, IntroParagraphs(test.summary))
		})
	}
}

func TestIntroFlattened(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		summary string
		want    string
	}{
		"Empty":            {summary: "", want: ""},
		"Single block":     {summary: "Just one line.", want: "Just one line."},
		"Joins with space": {summary: "First.\n\nSecond.\n\nThird.", want: "First. Second. Third."},
		"Trims blocks":     {summary: "  First.  \n\n  Second.  ", want: "First. Second."},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, IntroFlattened(test.summary))
		})
	}
}
