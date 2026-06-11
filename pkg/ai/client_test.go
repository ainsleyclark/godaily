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

	mockai "github.com/ainsleyclark/godaily/pkg/mocks/ai"
)

func TestClient_Prompt(t *testing.T) {
	t.Parallel()

	t.Run("Primary Success", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		primary := mockai.NewMockPrompter(ctrl)
		primary.EXPECT().Prompt(gomock.Any(), ModelSonnet, "sys", "user").Return([]byte("result"), nil)

		got, err := (&Client{primary: primary}).Prompt(context.Background(), ModelSonnet, "sys", "user")
		require.NoError(t, err)
		assert.Equal(t, []byte("result"), got)
	})

	t.Run("Strips Em Dashes From Response", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		primary := mockai.NewMockPrompter(ctrl)
		primary.EXPECT().
			Prompt(gomock.Any(), ModelOpus, "sys", "user").
			Return([]byte(`{"title":"Go 1.30 — out now","intro":"fast—really fast"}`), nil)

		got, err := (&Client{primary: primary}).Prompt(context.Background(), ModelOpus, "sys", "user")
		require.NoError(t, err)
		assert.Equal(t, `{"title":"Go 1.30 - out now","intro":"fast-really fast"}`, string(got))
		assert.NotContains(t, string(got), "—")
	})

	t.Run("Primary Fails", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		primary := mockai.NewMockPrompter(ctrl)
		primary.EXPECT().Prompt(gomock.Any(), ModelSonnet, "sys", "user").Return(nil, errors.New("primary failed"))

		_, err := (&Client{primary: primary}).Prompt(context.Background(), ModelSonnet, "sys", "user")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "primary failed")
	})
}
