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

	"github.com/ainsleyclark/godaily/pkg/news"
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
			assert.NotEmpty(t, got.ConfirmToken)
			assert.NotEmpty(t, got.UnsubscribeToken)
			assert.NotEqual(t, got.ConfirmToken, got.UnsubscribeToken)
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

	t.Run("FindByConfirmToken", func(t *testing.T) {
		t.Log("Happy path")
		{
			got, err := s.FindByConfirmToken(ctx, created.ConfirmToken)
			require.NoError(t, err)
			assert.Equal(t, created.ID, got.ID)
		}

		t.Log("Not found")
		{
			_, err := s.FindByConfirmToken(ctx, "missing")
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

	t.Run("Confirm", func(t *testing.T) {
		t.Log("Happy path")
		{
			require.NoError(t, s.Confirm(ctx, created.ConfirmToken))

			got, err := s.FindByEmail(ctx, "hello@example.com")
			require.NoError(t, err)
			require.NotNil(t, got.ConfirmedAt)
		}

		t.Log("Idempotent on already-confirmed token")
		{
			require.NoError(t, s.Confirm(ctx, created.ConfirmToken))
		}
	})

	t.Run("ListActive", func(t *testing.T) {
		got, err := s.ListActive(ctx)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, created.ID, got[0].ID)
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

		t.Log("FindByConfirmToken")
		{
			_, err := s.FindByConfirmToken(ctx, "x")
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

		t.Log("Confirm")
		{
			assert.Error(t, s.Confirm(ctx, "x"))
		}

		t.Log("Unsubscribe")
		{
			assert.Error(t, s.Unsubscribe(ctx, "x"))
		}

		t.Log("ListActive")
		{
			_, err := s.ListActive(ctx)
			assert.Error(t, err)
		}
	})
}
