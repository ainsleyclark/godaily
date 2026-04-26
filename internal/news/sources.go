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

package news

// Source defines a provider or source of information.
type Source string

// Source constants
const (
	SourceDevTo         Source = "dev_to"
	SourceGoBlog        Source = "go_blog"
	SourceGitHub        Source = "github"
	SourceReddit        Source = "reddit"
	SourceHN            Source = "hacker_news"
	SourceGolangBridge  Source = "golangbridge"
)

// Sources defines a list of all source types.
var Sources = []Source{
	SourceDevTo,
	SourceGoBlog,
	SourceGitHub,
	SourceReddit,
	SourceHN,
}

// String implements fmt.Stringer on source.
func (s Source) String() string {
	return string(s)
}
