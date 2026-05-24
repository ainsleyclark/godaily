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

	"github.com/ainsleyclark/godaily/pkg/env"
)

// Builder constructs a Fetcher from an env.Config. Sources register a
// Builder in init() so construction is deferred until configuration is
// loaded by env.New.
type Builder func(env.Config) Fetcher

var (
	registryMu sync.RWMutex
	registry   = map[Source]Builder{}
)

// Register associates a Source with a Builder.
// Called from each source package's init().
func Register(s Source, b Builder) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[s] = b
}

// Get returns the Fetcher for the given Source. Pre-Materialise the Builder
// runs against a zero env.Config, so callers depending on env-derived values
// must invoke Materialise during startup.
func Get(s Source) (Fetcher, error) {
	registryMu.RLock()
	b, ok := registry[s]
	registryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("no fetcher registered for source %q", s)
	}
	return b(env.Config{}), nil
}

// HasSources reports whether any source builders have been registered via
// init(). Bootstrap uses this to skip Materialise on Lambda functions that do
// not import pkg/source (i.e. every handler except /api/collect).
func HasSources() bool {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return len(registry) > 0
}

// Materialise builds every registered Source against cfg and replaces each
// Builder with a closure returning the prebuilt instance. Called from
// Bootstrap after env.New so .env values reach source constructors.
func Materialise(cfg env.Config) error {
	registryMu.Lock()
	defer registryMu.Unlock()
	for _, s := range Sources {
		b, ok := registry[s]
		if !ok {
			return fmt.Errorf("materialise: no builder registered for source %q", s)
		}
		f := b(cfg)
		registry[s] = func(env.Config) Fetcher { return f }
	}
	return nil
}

// SwapRegistry replaces the registry with fetchers from reg and returns a
// function that restores the previous registry. Each Fetcher is wrapped in
// a constant Builder. Intended for use in tests across packages.
func SwapRegistry(reg map[Source]Fetcher) (restore func()) {
	registryMu.Lock()
	defer registryMu.Unlock()
	orig := registry
	next := make(map[Source]Builder, len(reg))
	for s, f := range reg {
		next[s] = func(env.Config) Fetcher { return f }
	}
	registry = next
	return func() {
		registryMu.Lock()
		defer registryMu.Unlock()
		registry = orig
	}
}

// Validate checks that every entry in Sources has a registered builder.
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
