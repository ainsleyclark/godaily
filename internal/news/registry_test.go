package news

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegister(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		source  Source
		factory func() Fetcher
		want    func(error)
	}{
		"OK": {
			source:  "test_source",
			factory: func() Fetcher { return nil },
			want: func(err error) {
				assert.NoError(t, err)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			orig := registry
			registry = map[Source]func() Fetcher{}
			t.Cleanup(func() { registry = orig })

			Register(test.source, test.factory)
			_, err := Get(test.source)
			test.want(err)
		})
	}
}

func TestGet(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		setup  func()
		source Source
		want   func(Fetcher, error)
	}{
		"OK": {
			setup: func() {
				Register("test_get", func() Fetcher { return nil })
			},
			source: "test_get",
			want: func(_ Fetcher, err error) {
				assert.NoError(t, err)
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
			orig := registry
			registry = map[Source]func() Fetcher{}
			t.Cleanup(func() { registry = orig })

			test.setup()
			got, err := Get(test.source)
			test.want(got, err)
		})
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		setup func()
		want  func(error)
	}{
		"All Registered": {
			setup: func() {
				for _, s := range Sources {
					Register(s, func() Fetcher { return nil })
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
			orig := registry
			registry = map[Source]func() Fetcher{}
			t.Cleanup(func() { registry = orig })

			test.setup()
			err := Validate()
			test.want(err)
		})
	}
}
