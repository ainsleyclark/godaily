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
	"context"
	"testing"

	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubFetcher struct{}

func (stubFetcher) Fetch(context.Context) ([]Item, error) { return nil, nil }

func TestRegister(t *testing.T) {
	tt := map[string]struct {
		source  Source
		builder Builder
		want    func(error)
	}{
		"OK": {
			source:  "test_source",
			builder: func(env.Config) Fetcher { return stubFetcher{} },
			want: func(err error) {
				assert.NoError(t, err)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Cleanup(SwapRegistry(map[Source]Fetcher{}))

			Register(test.source, test.builder)
			_, err := Get(test.source)
			test.want(err)
		})
	}
}

func TestGet(t *testing.T) {
	tt := map[string]struct {
		setup  func()
		source Source
		want   func(Fetcher, error)
	}{
		"OK": {
			setup: func() {
				Register("test_get", func(env.Config) Fetcher { return stubFetcher{} })
			},
			source: "test_get",
			want: func(f Fetcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, f)
			},
		},
		"Not Found": {
			setup:  func() {},
			source: "unknown",
			want: func(f Fetcher, err error) {
				assert.Nil(t, f)
				assert.ErrorContains(t, err, "no fetcher registered for source")
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Cleanup(SwapRegistry(map[Source]Fetcher{}))

			test.setup()
			got, err := Get(test.source)
			test.want(got, err)
		})
	}
}

func TestValidate(t *testing.T) {
	tt := map[string]struct {
		setup func()
		want  func(error)
	}{
		"All Registered": {
			setup: func() {
				for _, s := range Sources {
					Register(s, func(env.Config) Fetcher { return stubFetcher{} })
				}
			},
			want: func(err error) {
				assert.NoError(t, err)
			},
		},
		"Missing Source": {
			setup: func() {},
			want: func(err error) {
				require.Error(t, err)
				assert.ErrorContains(t, err, "no fetcher registered for source")
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Cleanup(SwapRegistry(map[Source]Fetcher{}))

			test.setup()
			err := Validate()
			test.want(err)
		})
	}
}

// TestHasSources covers the HasSources() contract that guards both
// Materialise() in Bootstrap and Validate() in digest.New(). An empty registry
// (i.e. pkg/source was not imported) must return false so that non-collect
// Lambda functions can bootstrap without crashing.
func TestHasSources(t *testing.T) {
	t.Run("Empty registry returns false", func(t *testing.T) {
		t.Cleanup(SwapRegistry(map[Source]Fetcher{}))
		assert.False(t, HasSources())
	})

	t.Run("Non-empty registry returns true", func(t *testing.T) {
		t.Cleanup(SwapRegistry(map[Source]Fetcher{}))
		Register(SourceDevTo, func(env.Config) Fetcher { return stubFetcher{} })
		assert.True(t, HasSources())
	})
}

// TestHasSources_ValidateInvariant verifies the invariant that guards against
// the production crash where /api/social/featured and /api/social/rotation
// called Bootstrap() without importing pkg/source, causing news.Validate() to
// fail on SourceDevTo (the first entry in Sources) and os.Exit the process.
//
// The contract is: HasSources()==false iff the registry is empty, and callers
// (digest.New, Bootstrap) must skip Validate/Materialise in that case.
func TestHasSources_ValidateInvariant(t *testing.T) {
	t.Cleanup(SwapRegistry(map[Source]Fetcher{}))

	// No sources imported — simulates every Lambda except /api/collect.
	require.False(t, HasSources(), "empty registry must report HasSources=false")

	// Validate still returns an error for empty registry (its own contract
	// is unchanged); it is the CALLER's responsibility to check HasSources first.
	err := Validate()
	require.Error(t, err, "Validate must still error on empty registry so callers cannot forget the HasSources guard")
	assert.ErrorContains(t, err, string(SourceDevTo))
}

func TestMaterialise(t *testing.T) {
	t.Run("Builds Every Source", func(t *testing.T) {
		t.Cleanup(SwapRegistry(map[Source]Fetcher{}))

		calls := map[Source]int{}
		for _, s := range Sources {
			Register(s, func(env.Config) Fetcher {
				calls[s]++
				return stubFetcher{}
			})
		}

		require.NoError(t, Materialise(env.Config{}))
		for _, s := range Sources {
			assert.Equal(t, 1, calls[s], "builder for %s should run exactly once", s)
		}
	})

	t.Run("Missing Source Returns Error", func(t *testing.T) {
		t.Cleanup(SwapRegistry(map[Source]Fetcher{}))

		err := Materialise(env.Config{})
		require.Error(t, err)
		assert.ErrorContains(t, err, "materialise: no builder registered")
	})

	t.Run("Caches Built Fetchers", func(t *testing.T) {
		t.Cleanup(SwapRegistry(map[Source]Fetcher{}))

		calls := 0
		for _, s := range Sources {
			Register(s, func(env.Config) Fetcher {
				calls++
				return stubFetcher{}
			})
		}
		require.NoError(t, Materialise(env.Config{}))

		before := calls
		for _, s := range Sources {
			_, err := Get(s)
			require.NoError(t, err)
		}
		assert.Equal(t, before, calls, "Get must not re-invoke builders after Materialise")
	})
}
