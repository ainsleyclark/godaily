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

package env

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Setenv("APP_ENV", "production")

	t.Run("All vars set", func(t *testing.T) {
		t.Setenv("RESEND_TOKEN", "re_test")
		t.Setenv("ANTHROPIC_API_KEY", "sk-ant-test")
		t.Setenv("YOUTUBE_API_KEY", "yt_test")
		t.Setenv("GITHUB_TOKEN", "ghp_test")
		t.Setenv("EMAIL_SEND_ADDRESS", "test@example.com")
		t.Setenv("TURSO_URL", "file:./test.db")
		t.Setenv("TURSO_AUTH_TOKEN", "turso_test")

		cfg, err := New()

		require.NoError(t, err)
		assert.Equal(t, "re_test", cfg.ResendToken)
		assert.Equal(t, "sk-ant-test", cfg.AnthropicAPIKey)
		assert.Equal(t, "yt_test", cfg.YouTubeAPIKey)
		assert.Equal(t, "ghp_test", cfg.GitHubToken)
		assert.Equal(t, "test@example.com", cfg.EmailSendAddress)
		assert.Equal(t, "file:./test.db", cfg.TursoURL)
		assert.Equal(t, "turso_test", cfg.TursoAuthToken)
	})

	t.Run("Empty", func(t *testing.T) {
		t.Setenv("RESEND_TOKEN", "")
		t.Setenv("ANTHROPIC_API_KEY", "")
		t.Setenv("YOUTUBE_API_KEY", "")
		t.Setenv("GITHUB_TOKEN", "")
		t.Setenv("EMAIL_SEND_ADDRESS", "")
		t.Setenv("TURSO_URL", "")
		t.Setenv("TURSO_AUTH_TOKEN", "")

		cfg, err := New()

		require.NoError(t, err)
		assert.Empty(t, cfg.ResendToken)
		assert.Empty(t, cfg.AnthropicAPIKey)
		assert.Empty(t, cfg.YouTubeAPIKey)
		assert.Empty(t, cfg.GitHubToken)
		assert.Empty(t, cfg.EmailSendAddress)
		assert.Empty(t, cfg.TursoURL)
		assert.Empty(t, cfg.TursoAuthToken)
	})
}
