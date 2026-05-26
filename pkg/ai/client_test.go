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
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/mocks/ai"
	"github.com/ainsleyclark/godaily/pkg/mocks/slack"
)

func TestClient_Prompt(t *testing.T) {
	t.Parallel()

	t.Run("Primary Success", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		primary := mockai.NewMockPrompter(ctrl)
		fallback := mockai.NewMockPrompter(ctrl)
		primary.EXPECT().Prompt(gomock.Any(), "sys", "user").Return([]byte("result"), nil)
		fallback.EXPECT().Prompt(gomock.Any(), "sys", "user").Return([]byte("fallback"), nil)

		got, err := (&Client{primary: primary, fallback: fallback}).Prompt(context.Background(), "sys", "user")
		require.NoError(t, err)
		assert.Equal(t, []byte("result"), got)
	})

	t.Run("Posts Comparison To Slack", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		primary := mockai.NewMockPrompter(ctrl)
		fallback := mockai.NewMockPrompter(ctrl)
		slackMock := mockslack.NewMockSender(ctrl)
		primary.EXPECT().Prompt(gomock.Any(), "sys", "user").Return([]byte("anthropic out"), nil)
		fallback.EXPECT().Prompt(gomock.Any(), "sys", "user").Return([]byte("gemini out"), nil)

		var sent string
		slackMock.EXPECT().MustSend(gomock.Any(), gomock.Any()).Do(func(_ context.Context, msg string) {
			sent = msg
		})

		got, err := (&Client{primary: primary, fallback: fallback, notifier: slackMock}).Prompt(context.Background(), "sys", "user")
		require.NoError(t, err)
		assert.Equal(t, []byte("anthropic out"), got)
		assert.Contains(t, sent, "anthropic out")
		assert.Contains(t, sent, "gemini out")
	})

	t.Run("Primary Fails Nil Fallback", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		primary := mockai.NewMockPrompter(ctrl)
		primary.EXPECT().Prompt(gomock.Any(), "sys", "user").Return(nil, errors.New("primary failed"))

		_, err := (&Client{primary: primary}).Prompt(context.Background(), "sys", "user")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "primary failed")
	})

	t.Run("Primary Fails Fallback Used", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		primary := mockai.NewMockPrompter(ctrl)
		fallback := mockai.NewMockPrompter(ctrl)
		primary.EXPECT().Prompt(gomock.Any(), "sys", "user").Return(nil, errors.New("primary failed"))
		fallback.EXPECT().Prompt(gomock.Any(), "sys", "user").Return([]byte("fallback"), nil)

		got, err := (&Client{primary: primary, fallback: fallback}).Prompt(context.Background(), "sys", "user")
		require.NoError(t, err)
		assert.Equal(t, []byte("fallback"), got)
	})

	t.Run("Both Fail", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		primary := mockai.NewMockPrompter(ctrl)
		fallback := mockai.NewMockPrompter(ctrl)
		primary.EXPECT().Prompt(gomock.Any(), "sys", "user").Return(nil, errors.New("primary failed"))
		fallback.EXPECT().Prompt(gomock.Any(), "sys", "user").Return(nil, errors.New("fallback failed"))

		_, err := (&Client{primary: primary, fallback: fallback}).Prompt(context.Background(), "sys", "user")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fallback failed")
	})
}
