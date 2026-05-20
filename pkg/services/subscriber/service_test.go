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

package subscriber

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/gateway/email"
	mocknews "github.com/ainsleyclark/godaily/pkg/mocks/domain/news"
	"github.com/ainsleyclark/godaily/pkg/store"
)

// mockSender is a test double for email.Sender.
type mockSender struct {
	called bool
	req    email.SendEmailRequest
	err    error
}

func (m *mockSender) Send(_ context.Context, req email.SendEmailRequest) error {
	m.called = true
	m.req = req
	return m.err
}

func setup(t *testing.T) (
	*mocknews.MockSubscriberRepository,
	*mocknews.MockIssueRepository,
	*mockSender,
) {
	t.Helper()
	ctrl := gomock.NewController(t)
	return mocknews.NewMockSubscriberRepository(ctrl),
		mocknews.NewMockIssueRepository(ctrl),
		&mockSender{}
}

var errBoom = errors.New("boom")

func TestService_Subscribe(t *testing.T) {
	t.Parallel()

	sub := news.Subscriber{
		ID:               1,
		Email:            "user@example.com",
		UnsubscribeToken: "tok123",
		ConfirmToken:     "confirm-tok",
	}

	t.Run("Already Subscribed", func(t *testing.T) {
		t.Parallel()

		repo, issues, sender := setup(t)
		repo.EXPECT().FindByEmail(gomock.Any(), sub.Email).Return(sub, nil)

		_, err := New(repo, issues, sender).Subscribe(t.Context(), sub.Email)

		assert.ErrorIs(t, err, ErrAlreadySubscribed)
		assert.False(t, sender.called)
	})

	t.Run("FindByEmail Unexpected Error", func(t *testing.T) {
		t.Parallel()

		repo, issues, sender := setup(t)
		repo.EXPECT().FindByEmail(gomock.Any(), sub.Email).Return(news.Subscriber{}, errBoom)

		_, err := New(repo, issues, sender).Subscribe(t.Context(), sub.Email)

		assert.ErrorIs(t, err, errBoom)
		assert.False(t, sender.called)
	})

	t.Run("Missing Confirm Token", func(t *testing.T) {
		t.Parallel()

		repo, issues, sender := setup(t)
		repo.EXPECT().FindByEmail(gomock.Any(), sub.Email).Return(news.Subscriber{}, store.ErrNotFound)
		repo.EXPECT().Create(gomock.Any(), sub.Email).Return(news.Subscriber{Email: sub.Email}, nil)

		_, err := New(repo, issues, sender).Subscribe(t.Context(), sub.Email)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "confirmation token")
		assert.False(t, sender.called)
	})

	t.Run("Create Error", func(t *testing.T) {
		t.Parallel()

		repo, issues, sender := setup(t)
		repo.EXPECT().FindByEmail(gomock.Any(), sub.Email).Return(news.Subscriber{}, store.ErrNotFound)
		repo.EXPECT().Create(gomock.Any(), sub.Email).Return(news.Subscriber{}, errBoom)

		_, err := New(repo, issues, sender).Subscribe(t.Context(), sub.Email)

		assert.ErrorIs(t, err, errBoom)
		assert.False(t, sender.called)
	})

	t.Run("OK Sends Confirmation Email", func(t *testing.T) {
		t.Parallel()

		repo, issues, sender := setup(t)
		repo.EXPECT().FindByEmail(gomock.Any(), sub.Email).Return(news.Subscriber{}, store.ErrNotFound)
		repo.EXPECT().Create(gomock.Any(), sub.Email).Return(sub, nil)

		got, err := New(repo, issues, sender).Subscribe(t.Context(), sub.Email)

		require.NoError(t, err)
		assert.Equal(t, sub, got)
		assert.True(t, sender.called)
		assert.Equal(t, "Confirm your GoDaily subscription", sender.req.Subject)
		assert.Contains(t, sender.req.Html, sub.ConfirmToken)
		assert.Equal(t, "<https://godaily.dev/api/unsubscribe/?token=tok123>", sender.req.Headers["List-Unsubscribe"])
		assert.Equal(t, "List-Unsubscribe=One-Click", sender.req.Headers["List-Unsubscribe-Post"])
	})

	t.Run("Confirmation Email Failure Is Non Fatal", func(t *testing.T) {
		t.Parallel()

		repo, issues, sender := setup(t)
		sender.err = errBoom
		repo.EXPECT().FindByEmail(gomock.Any(), sub.Email).Return(news.Subscriber{}, store.ErrNotFound)
		repo.EXPECT().Create(gomock.Any(), sub.Email).Return(sub, nil)

		got, err := New(repo, issues, sender).Subscribe(t.Context(), sub.Email)

		require.NoError(t, err)
		assert.Equal(t, sub, got)
	})

	t.Run("Reactivate After Unsubscribe", func(t *testing.T) {
		t.Parallel()

		now := time.Now()
		unsubscribed := news.Subscriber{
			ID:               1,
			Email:            sub.Email,
			UnsubscribeToken: "old-token",
			UnsubscribedAt:   &now,
		}
		reactivated := news.Subscriber{
			ID:               1,
			Email:            sub.Email,
			UnsubscribeToken: "new-token",
			ConfirmToken:     "new-confirm-tok",
		}

		repo, issues, sender := setup(t)
		repo.EXPECT().FindByEmail(gomock.Any(), sub.Email).Return(unsubscribed, nil)
		repo.EXPECT().Reactivate(gomock.Any(), sub.Email).Return(reactivated, nil)

		got, err := New(repo, issues, sender).Subscribe(t.Context(), sub.Email)

		require.NoError(t, err)
		assert.Equal(t, reactivated, got)
		assert.True(t, sender.called)
		assert.Equal(t, "Confirm your GoDaily subscription", sender.req.Subject)
	})

	t.Run("Reactivate Error", func(t *testing.T) {
		t.Parallel()

		now := time.Now()
		unsubscribed := news.Subscriber{
			ID:             1,
			Email:          sub.Email,
			UnsubscribedAt: &now,
		}

		repo, issues, sender := setup(t)
		repo.EXPECT().FindByEmail(gomock.Any(), sub.Email).Return(unsubscribed, nil)
		repo.EXPECT().Reactivate(gomock.Any(), sub.Email).Return(news.Subscriber{}, errBoom)

		_, err := New(repo, issues, sender).Subscribe(t.Context(), sub.Email)

		assert.ErrorIs(t, err, errBoom)
		assert.False(t, sender.called)
	})
}

func TestService_Confirm(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		repo, issues, sender := setup(t)
		repo.EXPECT().Confirm(gomock.Any(), "confirm-tok").Return(news.Subscriber{}, nil)

		err := New(repo, issues, sender).Confirm(t.Context(), "confirm-tok")
		require.NoError(t, err)
	})

	t.Run("Invalid Token", func(t *testing.T) {
		t.Parallel()

		repo, issues, sender := setup(t)
		repo.EXPECT().Confirm(gomock.Any(), "bad-tok").Return(news.Subscriber{}, store.ErrNotFound)

		err := New(repo, issues, sender).Confirm(t.Context(), "bad-tok")
		assert.ErrorIs(t, err, store.ErrNotFound)
	})

	t.Run("Error", func(t *testing.T) {
		t.Parallel()

		repo, issues, sender := setup(t)
		repo.EXPECT().Confirm(gomock.Any(), "tok").Return(news.Subscriber{}, errBoom)

		err := New(repo, issues, sender).Confirm(t.Context(), "tok")
		assert.ErrorIs(t, err, errBoom)
	})
}

func TestService_Unsubscribe(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		repo, issues, sender := setup(t)
		repo.EXPECT().Unsubscribe(gomock.Any(), "tok123").Return(nil)

		err := New(repo, issues, sender).Unsubscribe(t.Context(), "tok123")
		require.NoError(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		t.Parallel()

		repo, issues, sender := setup(t)
		repo.EXPECT().Unsubscribe(gomock.Any(), "tok123").Return(errBoom)

		err := New(repo, issues, sender).Unsubscribe(t.Context(), "tok123")
		assert.ErrorIs(t, err, errBoom)
	})
}
