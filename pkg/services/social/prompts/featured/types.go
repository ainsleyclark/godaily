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

package featured

import (
	"errors"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
)

// Featured is the one item picked from the day's news to anchor every
// social post. Hook is the model's one-line reason this item matters,
// used to seed the per-platform reframing prompts.
type Featured struct {
	Title  string      `json:"title"`
	URL    string      `json:"url"`
	Source news.Source `json:"source"`
	Tag    news.Tag    `json:"tag"`
	Hook   string      `json:"hook"`
}

// ErrNoCandidates is returned by Feature when the input contains no
// items suitable for posting.
var ErrNoCandidates = errors.New("prompts: no candidate items")
