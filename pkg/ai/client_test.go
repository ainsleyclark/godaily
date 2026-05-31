// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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
		primary.EXPECT().Prompt(gomock.Any(), ModelSonnet, "sys", "user").Return([]byte("result"), nil)
		fallback.EXPECT().Prompt(gomock.Any(), ModelSonnet, "sys", "user").Return([]byte("fallback"), nil)

		got, err := (&Client{primary: primary, fallback: fallback}).Prompt(context.Background(), ModelSonnet, "sys", "user")
		require.NoError(t, err)
		assert.Equal(t, []byte("result"), got)
	})

	t.Run("Passes Model Through", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		primary := mockai.NewMockPrompter(ctrl)
		primary.EXPECT().Prompt(gomock.Any(), ModelOpus, "sys", "user").Return([]byte("result"), nil)

		got, err := (&Client{primary: primary}).Prompt(context.Background(), ModelOpus, "sys", "user")
		require.NoError(t, err)
		assert.Equal(t, []byte("result"), got)
	})

	t.Run("Posts Comparison To Slack", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		primary := mockai.NewMockPrompter(ctrl)
		fallback := mockai.NewMockPrompter(ctrl)
		slackMock := mockslack.NewMockSender(ctrl)
		primary.EXPECT().Prompt(gomock.Any(), ModelSonnet, "sys", "user").Return([]byte("anthropic out"), nil)
		fallback.EXPECT().Prompt(gomock.Any(), ModelSonnet, "sys", "user").Return([]byte("gemini out"), nil)

		var sent string
		slackMock.EXPECT().MustSend(gomock.Any(), gomock.Any()).Do(func(_ context.Context, msg string) {
			sent = msg
		})

		got, err := (&Client{primary: primary, fallback: fallback, notifier: slackMock}).Prompt(context.Background(), ModelSonnet, "sys", "user")
		require.NoError(t, err)
		assert.Equal(t, []byte("anthropic out"), got)
		assert.Contains(t, sent, "anthropic out")
		assert.Contains(t, sent, "gemini out")
	})

	t.Run("Primary Fails Nil Fallback", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		primary := mockai.NewMockPrompter(ctrl)
		primary.EXPECT().Prompt(gomock.Any(), ModelSonnet, "sys", "user").Return(nil, errors.New("primary failed"))

		_, err := (&Client{primary: primary}).Prompt(context.Background(), ModelSonnet, "sys", "user")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "primary failed")
	})

	t.Run("Primary Fails Fallback Used", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		primary := mockai.NewMockPrompter(ctrl)
		fallback := mockai.NewMockPrompter(ctrl)
		primary.EXPECT().Prompt(gomock.Any(), ModelSonnet, "sys", "user").Return(nil, errors.New("primary failed"))
		fallback.EXPECT().Prompt(gomock.Any(), ModelSonnet, "sys", "user").Return([]byte("fallback"), nil)

		got, err := (&Client{primary: primary, fallback: fallback}).Prompt(context.Background(), ModelSonnet, "sys", "user")
		require.NoError(t, err)
		assert.Equal(t, []byte("fallback"), got)
	})

	t.Run("Both Fail", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		primary := mockai.NewMockPrompter(ctrl)
		fallback := mockai.NewMockPrompter(ctrl)
		primary.EXPECT().Prompt(gomock.Any(), ModelSonnet, "sys", "user").Return(nil, errors.New("primary failed"))
		fallback.EXPECT().Prompt(gomock.Any(), ModelSonnet, "sys", "user").Return(nil, errors.New("fallback failed"))

		_, err := (&Client{primary: primary, fallback: fallback}).Prompt(context.Background(), ModelSonnet, "sys", "user")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fallback failed")
	})
}
