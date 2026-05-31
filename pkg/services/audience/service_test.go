// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package audience

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/audience"
	"github.com/ainsleyclark/godaily/pkg/gateway/email"
	"github.com/ainsleyclark/godaily/pkg/mocks/audience"
	"github.com/ainsleyclark/godaily/pkg/mocks/digest"
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
	*mockaudience.MockSubscriberRepository,
	*mockdigest.MockIssueRepository,
	*mockSender,
) {
	t.Helper()
	ctrl := gomock.NewController(t)
	return mockaudience.NewMockSubscriberRepository(ctrl),
		mockdigest.NewMockIssueRepository(ctrl),
		&mockSender{}
}

var errBoom = errors.New("boom")

func TestService_Subscribe(t *testing.T) {
	t.Parallel()

	sub := audience.Subscriber{
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

		assert.ErrorIs(t, err, audience.ErrAlreadySubscribed)
		assert.False(t, sender.called)
	})

	t.Run("FindByEmail Unexpected Error", func(t *testing.T) {
		t.Parallel()

		repo, issues, sender := setup(t)
		repo.EXPECT().FindByEmail(gomock.Any(), sub.Email).Return(audience.Subscriber{}, errBoom)

		_, err := New(repo, issues, sender).Subscribe(t.Context(), sub.Email)

		assert.ErrorIs(t, err, errBoom)
		assert.False(t, sender.called)
	})

	t.Run("Missing Confirm Token", func(t *testing.T) {
		t.Parallel()

		repo, issues, sender := setup(t)
		repo.EXPECT().FindByEmail(gomock.Any(), sub.Email).Return(audience.Subscriber{}, store.ErrNotFound)
		repo.EXPECT().Create(gomock.Any(), sub.Email).Return(audience.Subscriber{Email: sub.Email}, nil)

		_, err := New(repo, issues, sender).Subscribe(t.Context(), sub.Email)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "confirmation token")
		assert.False(t, sender.called)
	})

	t.Run("Create Error", func(t *testing.T) {
		t.Parallel()

		repo, issues, sender := setup(t)
		repo.EXPECT().FindByEmail(gomock.Any(), sub.Email).Return(audience.Subscriber{}, store.ErrNotFound)
		repo.EXPECT().Create(gomock.Any(), sub.Email).Return(audience.Subscriber{}, errBoom)

		_, err := New(repo, issues, sender).Subscribe(t.Context(), sub.Email)

		assert.ErrorIs(t, err, errBoom)
		assert.False(t, sender.called)
	})

	t.Run("OK Sends Confirmation Email", func(t *testing.T) {
		t.Parallel()

		repo, issues, sender := setup(t)
		repo.EXPECT().FindByEmail(gomock.Any(), sub.Email).Return(audience.Subscriber{}, store.ErrNotFound)
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
		repo.EXPECT().FindByEmail(gomock.Any(), sub.Email).Return(audience.Subscriber{}, store.ErrNotFound)
		repo.EXPECT().Create(gomock.Any(), sub.Email).Return(sub, nil)

		got, err := New(repo, issues, sender).Subscribe(t.Context(), sub.Email)

		require.NoError(t, err)
		assert.Equal(t, sub, got)
	})

	t.Run("Reactivate After Unsubscribe", func(t *testing.T) {
		t.Parallel()

		now := time.Now()
		unsubscribed := audience.Subscriber{
			ID:               1,
			Email:            sub.Email,
			UnsubscribeToken: "old-token",
			UnsubscribedAt:   &now,
		}
		reactivated := audience.Subscriber{
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
		unsubscribed := audience.Subscriber{
			ID:             1,
			Email:          sub.Email,
			UnsubscribedAt: &now,
		}

		repo, issues, sender := setup(t)
		repo.EXPECT().FindByEmail(gomock.Any(), sub.Email).Return(unsubscribed, nil)
		repo.EXPECT().Reactivate(gomock.Any(), sub.Email).Return(audience.Subscriber{}, errBoom)

		_, err := New(repo, issues, sender).Subscribe(t.Context(), sub.Email)

		assert.ErrorIs(t, err, errBoom)
		assert.False(t, sender.called)
	})
}

func TestService_SendConfirmationNudges(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)

	// newSvc builds a Service with a pinned clock so eligibility windows are
	// deterministic in tests.
	newSvc := func(repo *mockaudience.MockSubscriberRepository, issues *mockdigest.MockIssueRepository, sender *mockSender) *Service {
		s := New(repo, issues, sender)
		s.now = func() time.Time { return now }
		return s
	}

	// eligible is an unconfirmed sign-up squarely inside the nudge window.
	eligible := audience.Subscriber{
		ID:               1,
		Email:            "pending@example.com",
		UnsubscribeToken: "unsub-tok",
		ConfirmToken:     "confirm-tok",
		CreatedAt:        now.Add(-3 * 24 * time.Hour),
	}

	t.Run("OK Sends And Marks", func(t *testing.T) {
		t.Parallel()

		repo, issues, sender := setup(t)
		repo.EXPECT().List(gomock.Any(), store.ListOptions{}).Return([]audience.Subscriber{eligible}, nil)
		repo.EXPECT().MarkNudgeSent(gomock.Any(), eligible.ID).Return(nil)

		sent, failed, err := newSvc(repo, issues, sender).SendConfirmationNudges(t.Context())

		require.NoError(t, err)
		assert.Equal(t, 1, sent)
		assert.Equal(t, 0, failed)
		assert.True(t, sender.called)
		assert.Equal(t, "Confirm your GoDaily subscription, one click left", sender.req.Subject)
		assert.Contains(t, sender.req.Html, eligible.ConfirmToken)
		assert.Equal(t, "<https://godaily.dev/api/unsubscribe/?token=unsub-tok>", sender.req.Headers["List-Unsubscribe"])
	})

	t.Run("Skips Ineligible Subscribers", func(t *testing.T) {
		t.Parallel()

		confirmed := now.Add(-time.Hour)
		unsub := now.Add(-time.Hour)
		bounced := now.Add(-time.Hour)
		nudged := now.Add(-time.Hour)

		subs := []audience.Subscriber{
			{ID: 1, Email: "confirmed@example.com", ConfirmToken: "a", CreatedAt: now.Add(-3 * 24 * time.Hour), ConfirmedAt: &confirmed},
			{ID: 2, Email: "unsub@example.com", ConfirmToken: "b", CreatedAt: now.Add(-3 * 24 * time.Hour), UnsubscribedAt: &unsub},
			{ID: 3, Email: "bounced@example.com", ConfirmToken: "c", CreatedAt: now.Add(-3 * 24 * time.Hour), BouncedAt: &bounced},
			{ID: 4, Email: "nudged@example.com", ConfirmToken: "d", CreatedAt: now.Add(-3 * 24 * time.Hour), ConfirmationNudgeSentAt: &nudged},
			{ID: 5, Email: "no-token@example.com", CreatedAt: now.Add(-3 * 24 * time.Hour)},
			{ID: 6, Email: "too-fresh@example.com", ConfirmToken: "e", CreatedAt: now.Add(-2 * time.Hour)},
			{ID: 7, Email: "too-old@example.com", ConfirmToken: "f", CreatedAt: now.Add(-30 * 24 * time.Hour)},
		}

		repo, issues, sender := setup(t)
		repo.EXPECT().List(gomock.Any(), store.ListOptions{}).Return(subs, nil)

		sent, failed, err := newSvc(repo, issues, sender).SendConfirmationNudges(t.Context())

		require.NoError(t, err)
		assert.Equal(t, 0, sent)
		assert.Equal(t, 0, failed)
		assert.False(t, sender.called)
	})

	t.Run("Send Failure Counts As Failed And Does Not Mark", func(t *testing.T) {
		t.Parallel()

		repo, issues, sender := setup(t)
		sender.err = errBoom
		repo.EXPECT().List(gomock.Any(), store.ListOptions{}).Return([]audience.Subscriber{eligible}, nil)
		// MarkNudgeSent must NOT be called when the email failed, so the
		// subscriber stays eligible for a future run.

		sent, failed, err := newSvc(repo, issues, sender).SendConfirmationNudges(t.Context())

		require.NoError(t, err)
		assert.Equal(t, 0, sent)
		assert.Equal(t, 1, failed)
	})

	t.Run("List Error", func(t *testing.T) {
		t.Parallel()

		repo, issues, sender := setup(t)
		repo.EXPECT().List(gomock.Any(), store.ListOptions{}).Return(nil, errBoom)

		_, _, err := newSvc(repo, issues, sender).SendConfirmationNudges(t.Context())

		assert.ErrorIs(t, err, errBoom)
		assert.False(t, sender.called)
	})
}

func TestService_Confirm(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		repo, issues, sender := setup(t)
		repo.EXPECT().Confirm(gomock.Any(), "confirm-tok").Return(audience.Subscriber{}, nil)

		err := New(repo, issues, sender).Confirm(t.Context(), "confirm-tok")
		require.NoError(t, err)
	})

	t.Run("Invalid Token", func(t *testing.T) {
		t.Parallel()

		repo, issues, sender := setup(t)
		repo.EXPECT().Confirm(gomock.Any(), "bad-tok").Return(audience.Subscriber{}, store.ErrNotFound)

		err := New(repo, issues, sender).Confirm(t.Context(), "bad-tok")
		assert.ErrorIs(t, err, store.ErrNotFound)
	})

	t.Run("Error", func(t *testing.T) {
		t.Parallel()

		repo, issues, sender := setup(t)
		repo.EXPECT().Confirm(gomock.Any(), "tok").Return(audience.Subscriber{}, errBoom)

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

func TestService_MarkBounced(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		repo, issues, sender := setup(t)
		repo.EXPECT().MarkBounced(gomock.Any(), "user@example.com").Return(nil)

		err := New(repo, issues, sender).MarkBounced(t.Context(), "user@example.com")
		require.NoError(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		t.Parallel()

		repo, issues, sender := setup(t)
		repo.EXPECT().MarkBounced(gomock.Any(), "user@example.com").Return(errBoom)

		err := New(repo, issues, sender).MarkBounced(t.Context(), "user@example.com")
		assert.ErrorIs(t, err, errBoom)
	})
}

func TestService_MarkComplained(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		repo, issues, sender := setup(t)
		repo.EXPECT().MarkComplained(gomock.Any(), "user@example.com").Return(nil)

		err := New(repo, issues, sender).MarkComplained(t.Context(), "user@example.com")
		require.NoError(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		t.Parallel()

		repo, issues, sender := setup(t)
		repo.EXPECT().MarkComplained(gomock.Any(), "user@example.com").Return(errBoom)

		err := New(repo, issues, sender).MarkComplained(t.Context(), "user@example.com")
		assert.ErrorIs(t, err, errBoom)
	})
}
