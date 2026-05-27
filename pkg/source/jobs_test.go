// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasGoWord(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		in   string
		want bool
	}{
		"Capital Go":          {in: "Senior Go Engineer", want: true},
		"Lowercase go":        {in: "We use go for backend", want: true},
		"Golang":              {in: "Looking for a Golang dev", want: true},
		"go in punctuation":   {in: "Skills: Go, Python", want: true},
		"go-developer hyphen": {in: "Hiring Go-developer", want: true},
		"Mongo substring":     {in: "MongoDB experience preferred", want: false},
		"django substring":    {in: "Django background", want: false},
		"good not go":         {in: "Good communication skills", want: false},
		"Empty":               {in: "", want: false},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, hasGoWord(test.in))
		})
	}
}

func TestHasSalary(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		in   string
		want bool
	}{
		"Dollar amount":     {in: "$150k base", want: true},
		"Euro amount":       {in: "€80,000 annual", want: true},
		"Pound amount":      {in: "£100k", want: true},
		"Salary keyword":    {in: "Competitive salary offered", want: true},
		"Comp shorthand":    {in: "Comp is industry-leading", want: true},
		"Compensation word": {in: "Total compensation $200k", want: true},
		"Equity":            {in: "Generous equity package", want: true},
		"None":              {in: "We have a great culture", want: false},
		"Empty":             {in: "", want: false},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, hasSalary(test.in))
		})
	}
}

func TestIsRemote(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		in   string
		want bool
	}{
		"Remote":    {in: "Remote (EU only)", want: true},
		"Worldwide": {in: "Worldwide hiring", want: true},
		"Anywhere":  {in: "Work from anywhere", want: true},
		"Onsite":    {in: "Onsite in London", want: false},
		"Empty":     {in: "", want: false},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, isRemote(test.in))
		})
	}
}
