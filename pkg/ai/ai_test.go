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

package ai

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPrompter is a test double for Prompter.
type mockPrompter struct {
	out []byte
	err error
}

func (m *mockPrompter) Prompt(_ context.Context, _, _ string) ([]byte, error) {
	return m.out, m.err
}

func TestPrompt(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	primaryErr := errors.New("primary failed")
	fallbackErr := errors.New("fallback failed")

	t.Run("Primary Success", func(t *testing.T) {
		t.Parallel()
		primary := &mockPrompter{out: []byte("result")}
		fallback := &mockPrompter{out: []byte("fallback result")}

		got, err := prompt(ctx, primary, fallback, "sys", "user")
		require.NoError(t, err)
		assert.Equal(t, []byte("result"), got)
		// fallback should not have been called — we verify by checking that
		// if fallback had returned an error it would not appear.
	})

	t.Run("Primary Fails Nil Fallback", func(t *testing.T) {
		t.Parallel()
		primary := &mockPrompter{err: primaryErr}

		got, err := prompt(ctx, primary, nil, "sys", "user")
		require.ErrorIs(t, err, primaryErr)
		assert.Nil(t, got)
	})

	t.Run("Primary Fails Fallback Used", func(t *testing.T) {
		t.Parallel()
		primary := &mockPrompter{err: primaryErr}
		fallback := &mockPrompter{out: []byte("fallback result")}

		got, err := prompt(ctx, primary, fallback, "sys", "user")
		require.NoError(t, err)
		assert.Equal(t, []byte("fallback result"), got)
	})

	t.Run("Both Fail", func(t *testing.T) {
		t.Parallel()
		primary := &mockPrompter{err: primaryErr}
		fallback := &mockPrompter{err: fallbackErr}

		got, err := prompt(ctx, primary, fallback, "sys", "user")
		require.ErrorIs(t, err, fallbackErr)
		assert.Nil(t, got)
	})
}
