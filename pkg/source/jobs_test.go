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
