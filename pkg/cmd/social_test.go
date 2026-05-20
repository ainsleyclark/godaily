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

package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	socialgw "github.com/ainsleyclark/godaily/pkg/gateway/social"
)

func TestParsePlatforms(t *testing.T) {
	t.Parallel()

	t.Run("Empty input yields nil", func(t *testing.T) {
		t.Parallel()
		got, err := parsePlatforms(nil)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("All three platforms parsed in order", func(t *testing.T) {
		t.Parallel()
		got, err := parsePlatforms([]string{"bluesky", "linkedin", "mastodon"})
		require.NoError(t, err)
		assert.Equal(t, []socialgw.Platform{
			socialgw.PlatformBluesky,
			socialgw.PlatformLinkedIn,
			socialgw.PlatformMastodon,
		}, got)
	})

	t.Run("Whitespace + casing tolerated", func(t *testing.T) {
		t.Parallel()
		got, err := parsePlatforms([]string{"  Bluesky  ", "LINKEDIN"})
		require.NoError(t, err)
		assert.Equal(t, []socialgw.Platform{
			socialgw.PlatformBluesky,
			socialgw.PlatformLinkedIn,
		}, got)
	})

	t.Run("Unknown platform errors", func(t *testing.T) {
		t.Parallel()
		_, err := parsePlatforms([]string{"twitter"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "twitter")
	})
}
