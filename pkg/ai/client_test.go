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

type mockPrompter struct {
	called bool
	raw    []byte
	err    error
}

func (m *mockPrompter) Prompt(_ context.Context, _, _ string) ([]byte, error) {
	m.called = true
	return m.raw, m.err
}

func TestClient_Prompt(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	primaryErr := errors.New("primary failed")
	fallbackErr := errors.New("fallback failed")

	tt := map[string]struct {
		primary  *mockPrompter
		fallback *mockPrompter
		wantRaw  []byte
		wantErr  error
	}{
		"Primary Success": {
			primary:  &mockPrompter{raw: []byte("result")},
			fallback: &mockPrompter{raw: []byte("fallback result")},
			wantRaw:  []byte("result"),
		},
		"Primary Fails Nil Fallback": {
			primary: &mockPrompter{err: primaryErr},
			wantErr: primaryErr,
		},
		"Primary Fails Fallback Used": {
			primary:  &mockPrompter{err: primaryErr},
			fallback: &mockPrompter{raw: []byte("fallback result")},
			wantRaw:  []byte("fallback result"),
		},
		"Both Fail": {
			primary:  &mockPrompter{err: primaryErr},
			fallback: &mockPrompter{err: fallbackErr},
			wantErr:  fallbackErr,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var fallback Prompter
			if test.fallback != nil {
				fallback = test.fallback
			}
			c := New(test.primary, fallback)
			got, err := c.Prompt(ctx, "sys", "user")

			if test.wantErr != nil {
				require.ErrorIs(t, err, test.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.wantRaw, got)
			assert.True(t, test.primary.called)

			// When primary succeeds, fallback must not be called.
			if test.fallback != nil && test.primary.err == nil {
				assert.False(t, test.fallback.called, "fallback must not be called when primary succeeds")
			}
		})
	}
}
