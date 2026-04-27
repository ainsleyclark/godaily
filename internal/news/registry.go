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

import (
	"fmt"
	"sync"
)

var (
	registryMu sync.RWMutex
	registry   = map[Source]Fetcher{}
)

// Register associates a Source with a Fetcher.
// Called from each source package's init().
func Register(s Source, f Fetcher) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[s] = f
}

// Get returns the Fetcher for the given Source.
func Get(s Source) (Fetcher, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	f, ok := registry[s]
	if !ok {
		return nil, fmt.Errorf("no fetcher registered for source %q", s)
	}
	return f, nil
}

// Validate checks that every entry in Sources has a registered fetcher.
// Call at startup or in tests to catch missing registrations early.
func Validate() error {
	registryMu.RLock()
	defer registryMu.RUnlock()
	for _, s := range Sources {
		if _, ok := registry[s]; !ok {
			return fmt.Errorf("no fetcher registered for source %q", s)
		}
	}
	return nil
}
