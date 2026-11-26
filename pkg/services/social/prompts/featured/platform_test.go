// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package featured

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/mocks/ai"
	"github.com/ainsleyclark/godaily/pkg/services/social/compose"
)

// footerBudget is the body budget the prompt advertises once the
// featured Subscribe footer is reserved.
func footerBudget(p social.Platform) int {
	return compose.Limit(p) - compose.Length(p, compose.SubscribeLine(p, "featured")) - 2
}

func sampleFeatured() Featured {
	return Featured{
		Title:  "Go 1.30 released",
		URL:    "https://go.dev/blog/go1.30",
		Source: news.SourceGoRelease,
		Tag:    news.TagRelease,
		Hook:   "Go 1.30 ships generic type inference improvements.",
	}
}

func TestBuildPlatformSystem(t *testing.T) {
	t.Parallel()

	cfg := platformConfig{
		name:      "Bluesky",
		charLimit: 300,
		hashtags:  []string{"#golang"},
		guidance:  "be terse",
	}
	sys := buildPlatformSystem(cfg, cfg.charLimit)

	assert.Contains(t, sys, "Bluesky")
	assert.Contains(t, sys, "300 characters")
	assert.Contains(t, sys, "#golang")
	assert.Contains(t, sys, "be terse")
	assert.Contains(t, sys, "JSON")
	assert.NotContains(t, sys, "No hashtags",
		"no-hashtag branch should not appear when hashtags are present")
}

func TestBuildPlatformSystem_NoHashtags(t *testing.T) {
	t.Parallel()

	sys := buildPlatformSystem(platformConfig{name: "X", charLimit: 100, guidance: "g"}, 100)
	assert.Contains(t, sys, "No hashtags")
}

func TestReframe(t *testing.T) {
	t.Parallel()

	cfg := platformConfig{
		name:      "Bluesky",
		charLimit: 300,
		hashtags:  []string{"#golang"},
		guidance:  "Be terse.",
	}

	t.Run("Nil prompter errors", func(t *testing.T) {
		t.Parallel()
		_, err := reframe(t.Context(), nil, cfg, sampleFeatured())
		require.Error(t, err)
	})

	t.Run("Empty URL errors", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		p := mockai.NewMockPrompter(ctrl)
		f := sampleFeatured()
		f.URL = ""
		_, err := reframe(t.Context(), p, cfg, f)
		require.Error(t, err)
	})

	t.Run("Happy path returns trimmed text", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		p := mockai.NewMockPrompter(ctrl)
		p.EXPECT().
			Prompt(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return([]byte(`{"text":"  Go 1.30 ships generic improvements.\n\nhttps://go.dev/blog/go1.30\n#golang  "}`), nil)

		got, err := reframe(t.Context(), p, cfg, sampleFeatured())
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(got, "Go 1.30"))
		assert.Contains(t, got, "https://go.dev/blog/go1.30")
		assert.Contains(t, got, "#golang")
	})

	t.Run("Strips fences", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		p := mockai.NewMockPrompter(ctrl)
		fenced := "```json\n" + `{"text":"hello"}` + "\n```"
		p.EXPECT().Prompt(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]byte(fenced), nil)

		got, err := reframe(t.Context(), p, cfg, sampleFeatured())
		require.NoError(t, err)
		assert.Equal(t, "hello", got)
	})

	t.Run("AI error wrapped", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		p := mockai.NewMockPrompter(ctrl)
		p.EXPECT().Prompt(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("network down"))

		_, err := reframe(t.Context(), p, cfg, sampleFeatured())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ai")
	})

	t.Run("Empty body errors", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		p := mockai.NewMockPrompter(ctrl)
		p.EXPECT().Prompt(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]byte("   "), nil)

		_, err := reframe(t.Context(), p, cfg, sampleFeatured())
		require.Error(t, err)
	})

	t.Run("Empty text field errors", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		p := mockai.NewMockPrompter(ctrl)
		p.EXPECT().Prompt(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]byte(`{"text":""}`), nil)

		_, err := reframe(t.Context(), p, cfg, sampleFeatured())
		require.Error(t, err)
	})

	t.Run("Over-limit text is truncated to the char limit", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		p := mockai.NewMockPrompter(ctrl)
		tiny := platformConfig{name: "X", charLimit: 5, hashtags: nil, guidance: "g"}
		p.EXPECT().Prompt(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return([]byte(`{"text":"this is well over five characters"}`), nil)

		got, err := reframe(t.Context(), p, tiny, sampleFeatured())
		require.NoError(t, err)
		assert.LessOrEqual(t, utf8.RuneCountInString(got), tiny.charLimit,
			"over-limit text must be truncated to the char limit")
	})
}

func TestBlueskyLinkedInMastodonShape(t *testing.T) {
	t.Parallel()

	t.Run("Bluesky calls reframe with #golang and the 300 limit less the footer", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		p := mockai.NewMockPrompter(ctrl)
		p.EXPECT().
			Prompt(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ any, _, system, _ string) ([]byte, error) {
				assert.Contains(t, system, "Bluesky")
				assert.Contains(t, system, fmt.Sprintf("%d characters", footerBudget(social.Bluesky)))
				assert.Contains(t, system, "#golang")
				return []byte(`{"text":"ok"}`), nil
			})

		_, err := Bluesky(t.Context(), p, sampleFeatured())
		require.NoError(t, err)
	})

	t.Run("LinkedIn uses the 1300 limit less the footer and 3 hashtags", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		p := mockai.NewMockPrompter(ctrl)
		p.EXPECT().
			Prompt(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ any, _, system, _ string) ([]byte, error) {
				assert.Contains(t, system, "LinkedIn")
				assert.Contains(t, system, fmt.Sprintf("%d characters", footerBudget(social.LinkedIn)))
				for _, tag := range LinkedInHashtags {
					assert.Contains(t, system, tag)
				}
				return []byte(`{"text":"ok"}`), nil
			})

		_, err := LinkedIn(t.Context(), p, sampleFeatured())
		require.NoError(t, err)
	})

	t.Run("Mastodon uses the 500 limit less the footer and #go", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		p := mockai.NewMockPrompter(ctrl)
		p.EXPECT().
			Prompt(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ any, _, system, _ string) ([]byte, error) {
				assert.Contains(t, system, "Mastodon")
				assert.Contains(t, system, fmt.Sprintf("%d characters", footerBudget(social.Mastodon)))
				assert.Contains(t, system, "#go")
				return []byte(`{"text":"ok"}`), nil
			})

		_, err := Mastodon(t.Context(), p, sampleFeatured())
		require.NoError(t, err)
	})
}

func TestReframe_SubscribeFooter(t *testing.T) {
	t.Parallel()

	for _, plat := range []social.Platform{social.Bluesky, social.LinkedIn, social.Mastodon} {
		t.Run(plat.String(), func(t *testing.T) {
			t.Parallel()

			cfg := platformConfig{
				platform:  plat,
				name:      plat.String(),
				charLimit: compose.Limit(plat),
				hashtags:  []string{"#golang"},
				guidance:  "g",
			}

			t.Run("Links back to godaily.dev with featured UTM params", func(t *testing.T) {
				t.Parallel()
				ctrl := gomock.NewController(t)
				p := mockai.NewMockPrompter(ctrl)
				p.EXPECT().Prompt(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]byte(`{"text":"Go 1.30 ships.\n\nhttps://go.dev/blog/go1.30\n#golang"}`), nil)

				got, err := reframe(t.Context(), p, cfg, sampleFeatured())
				require.NoError(t, err)
				assert.True(t, strings.HasSuffix(got, "\n\n"+compose.SubscribeLine(plat, "featured")))
				assert.Contains(t, got, "utm_campaign=featured")
				assert.Contains(t, got, "utm_source=social-"+plat.String())
			})

			t.Run("Over-long body is trimmed and the footer survives", func(t *testing.T) {
				t.Parallel()
				ctrl := gomock.NewController(t)
				p := mockai.NewMockPrompter(ctrl)
				long := strings.Repeat("Go 1.30 ships a lot of things. ", 60)
				p.EXPECT().Prompt(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]byte(`{"text":"`+long+`\n\nhttps://go.dev/blog/go1.30\n#golang"}`), nil)

				got, err := reframe(t.Context(), p, cfg, sampleFeatured())
				require.NoError(t, err)
				assert.LessOrEqual(t, compose.Length(plat, got), compose.Limit(plat))
				assert.Contains(t, got, compose.SubscribeLine(plat, "featured"))
			})
		})
	}
}
