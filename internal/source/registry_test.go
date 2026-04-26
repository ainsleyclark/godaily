package source

import (
	"testing"

	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/stretchr/testify/assert"
)

func TestRegisteredSources(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		source news.Source
		want   func(news.Fetcher, error)
	}{
		"DevTo": {
			source: news.SourceDevTo,
			want: func(f news.Fetcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, f)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			got, err := news.Get(test.source)
			test.want(got, err)
		})
	}
}
