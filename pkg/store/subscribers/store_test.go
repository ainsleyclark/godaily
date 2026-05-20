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

package subscribers_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/store"
	"github.com/ainsleyclark/godaily/pkg/store/internal/dbtest"
	"github.com/ainsleyclark/godaily/pkg/store/subscribers"
)

func TestSubscribers_Store(t *testing.T) {
	ctx, db, teardown := dbtest.Setup(t)
	defer teardown()
	s := subscribers.New(db)

	var created news.Subscriber

	t.Run("Create", func(t *testing.T) {
		t.Log("Normalises email and generates tokens")
		{
			got, err := s.Create(ctx, "  Hello@Example.COM  ")
			require.NoError(t, err)
			assert.NotZero(t, got.ID)
			assert.Equal(t, "hello@example.com", got.Email)
			assert.NotEmpty(t, got.UnsubscribeToken)
			assert.NotEmpty(t, got.ConfirmToken)
			assert.Nil(t, got.ConfirmedAt)
			assert.Nil(t, got.UnsubscribedAt)
			assert.False(t, got.CreatedAt.IsZero())
			created = got
		}

		t.Log("Rejects empty email")
		{
			_, err := s.Create(ctx, "   ")
			assert.Error(t, err)
		}

		t.Log("Rejects duplicate email")
		{
			_, err := s.Create(ctx, "hello@example.com")
			assert.Error(t, err)
		}
	})

	t.Run("Find", func(t *testing.T) {
		t.Log("Happy path")
		{
			got, err := s.Find(ctx, created.ID)
			require.NoError(t, err)
			assert.Equal(t, created.ID, got.ID)
			assert.Equal(t, created.Email, got.Email)
		}

		t.Log("Not found")
		{
			_, err := s.Find(ctx, 999_999)
			require.Error(t, err)
			assert.Equal(t, store.ErrNotFound, err)
		}
	})

	t.Run("FindByEmail", func(t *testing.T) {
		t.Log("Happy path")
		{
			got, err := s.FindByEmail(ctx, "hello@example.com")
			require.NoError(t, err)
			assert.Equal(t, created.ID, got.ID)
		}

		t.Log("Not found")
		{
			_, err := s.FindByEmail(ctx, "missing@example.com")
			require.Error(t, err)
			assert.Equal(t, store.ErrNotFound, err)
		}
	})

	t.Run("FindByUnsubscribeToken", func(t *testing.T) {
		t.Log("Happy path")
		{
			got, err := s.FindByUnsubscribeToken(ctx, created.UnsubscribeToken)
			require.NoError(t, err)
			assert.Equal(t, created.ID, got.ID)
		}

		t.Log("Not found")
		{
			_, err := s.FindByUnsubscribeToken(ctx, "missing")
			require.Error(t, err)
			assert.Equal(t, store.ErrNotFound, err)
		}
	})

	t.Run("ListActive", func(t *testing.T) {
		t.Log("Excludes unconfirmed subscriber")
		{
			got, err := s.ListActive(ctx)
			require.NoError(t, err)
			assert.Empty(t, got)
		}
	})

	t.Run("Confirm", func(t *testing.T) {
		t.Log("Happy path sets confirmed_at and clears confirm_token")
		{
			got, err := s.Confirm(ctx, created.ConfirmToken)
			require.NoError(t, err)
			assert.Equal(t, created.ID, got.ID)
			assert.NotNil(t, got.ConfirmedAt)
			assert.Empty(t, got.ConfirmToken)
			created = got
		}

		t.Log("Invalid or already-used token returns not found")
		{
			_, err := s.Confirm(ctx, "bad-token")
			assert.ErrorIs(t, err, store.ErrNotFound)
		}
	})

	t.Run("ListActive After Confirm", func(t *testing.T) {
		t.Log("Returns confirmed subscriber")
		{
			got, err := s.ListActive(ctx)
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, created.ID, got[0].ID)
		}
	})

	t.Run("CountActive", func(t *testing.T) {
		count, err := s.CountActive(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})

	t.Run("Unsubscribe", func(t *testing.T) {
		t.Log("Happy path")
		{
			require.NoError(t, s.Unsubscribe(ctx, created.UnsubscribeToken))

			got, err := s.FindByEmail(ctx, "hello@example.com")
			require.NoError(t, err)
			require.NotNil(t, got.UnsubscribedAt)
		}

		t.Log("Idempotent on already-unsubscribed token")
		{
			require.NoError(t, s.Unsubscribe(ctx, created.UnsubscribeToken))
		}

		t.Log("Removes subscriber from ListActive")
		{
			active, err := s.ListActive(ctx)
			require.NoError(t, err)
			assert.Empty(t, active)
		}
	})

	t.Run("Reactivate", func(t *testing.T) {
		t.Log("Happy path resets confirmed_at and sets new confirm_token")
		{
			got, err := s.Reactivate(ctx, "hello@example.com")
			require.NoError(t, err)
			assert.Equal(t, created.ID, got.ID)
			assert.Equal(t, "hello@example.com", got.Email)
			assert.Nil(t, got.UnsubscribedAt)
			assert.Nil(t, got.ConfirmedAt)
			assert.NotEmpty(t, got.UnsubscribeToken)
			assert.NotEmpty(t, got.ConfirmToken)
			assert.NotEqual(t, created.UnsubscribeToken, got.UnsubscribeToken)
			created = got
		}

		t.Log("Not in ListActive until confirmed again")
		{
			active, err := s.ListActive(ctx)
			require.NoError(t, err)
			assert.Empty(t, active)
		}

		t.Log("Not found for non-existent or already-active email")
		{
			_, err := s.Reactivate(ctx, "missing@example.com")
			assert.ErrorIs(t, err, store.ErrNotFound)
		}
	})

	t.Run("MarkBounced", func(t *testing.T) {
		bouncer, err := s.Create(ctx, "bounce@example.com")
		require.NoError(t, err)
		_, err = s.Confirm(ctx, bouncer.ConfirmToken)
		require.NoError(t, err)

		t.Log("Sets bounced_at")
		{
			require.NoError(t, s.MarkBounced(ctx, "bounce@example.com"))
			got, err := s.FindByEmail(ctx, "bounce@example.com")
			require.NoError(t, err)
			assert.NotNil(t, got.BouncedAt)
		}

		t.Log("Idempotent on an already-bounced address")
		{
			require.NoError(t, s.MarkBounced(ctx, "bounce@example.com"))
		}

		t.Log("Excluded from ListActive")
		{
			active, err := s.ListActive(ctx)
			require.NoError(t, err)
			assert.Empty(t, active)
		}

		t.Log("Unknown email is a no-op")
		{
			require.NoError(t, s.MarkBounced(ctx, "ghost@example.com"))
		}
	})

	t.Run("MarkComplained", func(t *testing.T) {
		complainer, err := s.Create(ctx, "spam@example.com")
		require.NoError(t, err)
		_, err = s.Confirm(ctx, complainer.ConfirmToken)
		require.NoError(t, err)

		t.Log("Sets unsubscribed_at")
		{
			require.NoError(t, s.MarkComplained(ctx, "spam@example.com"))
			got, err := s.FindByEmail(ctx, "spam@example.com")
			require.NoError(t, err)
			assert.NotNil(t, got.UnsubscribedAt)
		}

		t.Log("Idempotent on an already-unsubscribed address")
		{
			require.NoError(t, s.MarkComplained(ctx, "spam@example.com"))
		}
	})

	// MUST be last: closing the DB makes every subsequent query fail.
	t.Run("Query Error On Closed DB", func(t *testing.T) {
		require.NoError(t, db.Close())

		t.Log("Find")
		{
			_, err := s.Find(ctx, 1)
			assert.Error(t, err)
			assert.NotErrorIs(t, err, store.ErrNotFound)
		}

		t.Log("FindByEmail")
		{
			_, err := s.FindByEmail(ctx, "x")
			assert.Error(t, err)
		}

		t.Log("FindByUnsubscribeToken")
		{
			_, err := s.FindByUnsubscribeToken(ctx, "x")
			assert.Error(t, err)
		}

		t.Log("Create")
		{
			_, err := s.Create(ctx, "x@example.com")
			assert.Error(t, err)
		}

		t.Log("Reactivate")
		{
			_, err := s.Reactivate(ctx, "x@example.com")
			assert.Error(t, err)
		}

		t.Log("Confirm")
		{
			_, err := s.Confirm(ctx, "tok")
			assert.Error(t, err)
		}

		t.Log("Unsubscribe")
		{
			assert.Error(t, s.Unsubscribe(ctx, "x"))
		}

		t.Log("MarkBounced")
		{
			assert.Error(t, s.MarkBounced(ctx, "x@example.com"))
		}

		t.Log("MarkComplained")
		{
			assert.Error(t, s.MarkComplained(ctx, "x@example.com"))
		}

		t.Log("ListActive")
		{
			_, err := s.ListActive(ctx)
			assert.Error(t, err)
		}

		t.Log("CountActive")
		{
			_, err := s.CountActive(ctx)
			assert.Error(t, err)
		}
	})
}
