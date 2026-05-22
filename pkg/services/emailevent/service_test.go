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

package emailevent_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	mockengagement "github.com/ainsleyclark/godaily/pkg/mocks/domain/engagement"
	mocksubscriber "github.com/ainsleyclark/godaily/pkg/mocks/subscriber"
	"github.com/ainsleyclark/godaily/pkg/services/emailevent"
)

var errBoom = errors.New("boom")

// stubItemFinder is a controllable emailevent.ItemFinder for tests. The zero
// value resolves nothing, mirroring a click that matches no item.
type stubItemFinder struct {
	id  int64
	ok  bool
	err error
}

func (s stubItemFinder) FindByURLInIssue(context.Context, int64, string) (int64, bool, error) {
	return s.id, s.ok, s.err
}

func setup(t *testing.T) (*mockengagement.MockEmailEventRepository, *mocksubscriber.MockSubscriber, *emailevent.Service) {
	t.Helper()
	ctrl := gomock.NewController(t)
	events := mockengagement.NewMockEmailEventRepository(ctrl)
	subs := mocksubscriber.NewMockSubscriber(ctrl)
	return events, subs, emailevent.New(events, subs, stubItemFinder{}, "admin@example.com")
}

func TestService_Process(t *testing.T) {
	t.Parallel()

	opened := engagement.EmailEvent{Type: engagement.EmailEventTypeOpened, EventID: "evt_opened", Email: "reader@example.com"}

	t.Run("Stores event with no side effect", func(t *testing.T) {
		t.Parallel()

		events, _, svc := setup(t)
		events.EXPECT().ExistsByEventID(gomock.Any(), "evt_opened").Return(false, nil)
		events.EXPECT().Create(gomock.Any(), opened).Return(opened, nil)

		require.NoError(t, svc.Process(t.Context(), opened))
	})

	t.Run("Bounced event marks the subscriber bounced", func(t *testing.T) {
		t.Parallel()

		bounced := engagement.EmailEvent{Type: engagement.EmailEventTypeBounced, EventID: "evt_bounced", Email: "dead@example.com"}
		events, subs, svc := setup(t)
		events.EXPECT().ExistsByEventID(gomock.Any(), "evt_bounced").Return(false, nil)
		events.EXPECT().Create(gomock.Any(), bounced).Return(bounced, nil)
		subs.EXPECT().MarkBounced(gomock.Any(), "dead@example.com").Return(nil)

		require.NoError(t, svc.Process(t.Context(), bounced))
	})

	t.Run("Complained event unsubscribes the subscriber", func(t *testing.T) {
		t.Parallel()

		complained := engagement.EmailEvent{Type: engagement.EmailEventTypeComplained, EventID: "evt_spam", Email: "angry@example.com"}
		events, subs, svc := setup(t)
		events.EXPECT().ExistsByEventID(gomock.Any(), "evt_spam").Return(false, nil)
		events.EXPECT().Create(gomock.Any(), complained).Return(complained, nil)
		subs.EXPECT().MarkComplained(gomock.Any(), "angry@example.com").Return(nil)

		require.NoError(t, svc.Process(t.Context(), complained))
	})

	t.Run("Duplicate event is skipped", func(t *testing.T) {
		t.Parallel()

		events, _, svc := setup(t)
		events.EXPECT().ExistsByEventID(gomock.Any(), "evt_opened").Return(true, nil)

		require.NoError(t, svc.Process(t.Context(), opened))
	})

	t.Run("Existence check error propagates", func(t *testing.T) {
		t.Parallel()

		events, _, svc := setup(t)
		events.EXPECT().ExistsByEventID(gomock.Any(), "evt_opened").Return(false, errBoom)

		assert.ErrorIs(t, svc.Process(t.Context(), opened), errBoom)
	})

	t.Run("Store error propagates", func(t *testing.T) {
		t.Parallel()

		events, _, svc := setup(t)
		events.EXPECT().ExistsByEventID(gomock.Any(), "evt_opened").Return(false, nil)
		events.EXPECT().Create(gomock.Any(), opened).Return(engagement.EmailEvent{}, errBoom)

		assert.ErrorIs(t, svc.Process(t.Context(), opened), errBoom)
	})

	t.Run("Side effect error propagates", func(t *testing.T) {
		t.Parallel()

		bounced := engagement.EmailEvent{Type: engagement.EmailEventTypeBounced, EventID: "evt_bounced", Email: "dead@example.com"}
		events, subs, svc := setup(t)
		events.EXPECT().ExistsByEventID(gomock.Any(), "evt_bounced").Return(false, nil)
		events.EXPECT().Create(gomock.Any(), bounced).Return(bounced, nil)
		subs.EXPECT().MarkBounced(gomock.Any(), "dead@example.com").Return(errBoom)

		assert.ErrorIs(t, svc.Process(t.Context(), bounced), errBoom)
	})

	t.Run("Admin email is silently ignored", func(t *testing.T) {
		t.Parallel()

		evt := engagement.EmailEvent{Type: engagement.EmailEventTypeOpened, EventID: "evt_admin", Email: "admin@example.com"}
		_, _, svc := setup(t)

		require.NoError(t, svc.Process(t.Context(), evt))
	})

	t.Run("Admin email matching is case-insensitive", func(t *testing.T) {
		t.Parallel()

		evt := engagement.EmailEvent{Type: engagement.EmailEventTypeOpened, EventID: "evt_admin", Email: "Admin@Example.com"}
		_, _, svc := setup(t)

		require.NoError(t, svc.Process(t.Context(), evt))
	})

	t.Run("godaily.dev address is silently ignored", func(t *testing.T) {
		t.Parallel()

		evt := engagement.EmailEvent{Type: engagement.EmailEventTypeOpened, EventID: "evt_internal", Email: "hello@godaily.dev"}
		_, _, svc := setup(t)

		require.NoError(t, svc.Process(t.Context(), evt))
	})
}

func TestService_Process_ClickItemResolution(t *testing.T) {
	t.Parallel()

	issueID := int64(7)
	itemID := int64(42)
	click := engagement.EmailEvent{
		Type:    engagement.EmailEventTypeClicked,
		EventID: "evt_click",
		Email:   "reader@example.com",
		IssueID: &issueID,
		URL:     "https://example.com/article",
	}

	newSvc := func(t *testing.T, finder emailevent.ItemFinder) (*mockengagement.MockEmailEventRepository, *emailevent.Service) {
		t.Helper()
		ctrl := gomock.NewController(t)
		events := mockengagement.NewMockEmailEventRepository(ctrl)
		subs := mocksubscriber.NewMockSubscriber(ctrl)
		return events, emailevent.New(events, subs, finder, "admin@example.com")
	}

	t.Run("Resolved item is stored on the event", func(t *testing.T) {
		t.Parallel()

		events, svc := newSvc(t, stubItemFinder{id: itemID, ok: true})
		events.EXPECT().ExistsByEventID(gomock.Any(), "evt_click").Return(false, nil)

		want := click
		want.ItemID = &itemID
		events.EXPECT().Create(gomock.Any(), want).Return(want, nil)

		require.NoError(t, svc.Process(t.Context(), click))
	})

	t.Run("Click matching no item stores a nil item", func(t *testing.T) {
		t.Parallel()

		events, svc := newSvc(t, stubItemFinder{})
		events.EXPECT().ExistsByEventID(gomock.Any(), "evt_click").Return(false, nil)
		events.EXPECT().Create(gomock.Any(), click).Return(click, nil)

		require.NoError(t, svc.Process(t.Context(), click))
	})

	t.Run("Lookup error does not fail the event", func(t *testing.T) {
		t.Parallel()

		events, svc := newSvc(t, stubItemFinder{err: errBoom})
		events.EXPECT().ExistsByEventID(gomock.Any(), "evt_click").Return(false, nil)
		events.EXPECT().Create(gomock.Any(), click).Return(click, nil)

		require.NoError(t, svc.Process(t.Context(), click))
	})
}
