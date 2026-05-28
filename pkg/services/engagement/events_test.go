// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package engagement_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/mocks/audience"
	"github.com/ainsleyclark/godaily/pkg/mocks/engagement"
	engagementsvc "github.com/ainsleyclark/godaily/pkg/services/engagement"
)

var errBoom = errors.New("boom")

// stubItemFinder is a controllable engagement.ItemFinder for tests. The zero
// value resolves nothing, mirroring a click that matches no item.
type stubItemFinder struct {
	id  int64
	ok  bool
	err error
}

func (s stubItemFinder) FindByURLInIssue(context.Context, int64, string) (int64, bool, error) {
	return s.id, s.ok, s.err
}

func setup(t *testing.T) (*mockengagement.MockEmailEventRepository, *mockaudience.MockSubscriberService, *engagementsvc.EventService) {
	t.Helper()
	ctrl := gomock.NewController(t)
	events := mockengagement.NewMockEmailEventRepository(ctrl)
	subs := mockaudience.NewMockSubscriberService(ctrl)
	return events, subs, engagementsvc.NewEvents(events, subs, stubItemFinder{}, "admin@example.com")
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

	t.Run("Suppressed event marks the subscriber suppressed", func(t *testing.T) {
		t.Parallel()

		suppressed := engagement.EmailEvent{Type: engagement.EmailEventTypeSuppressed, EventID: "evt_suppressed", Email: "suppressed@example.com"}
		events, subs, svc := setup(t)
		events.EXPECT().ExistsByEventID(gomock.Any(), "evt_suppressed").Return(false, nil)
		events.EXPECT().Create(gomock.Any(), suppressed).Return(suppressed, nil)
		subs.EXPECT().MarkSuppressed(gomock.Any(), "suppressed@example.com").Return(nil)

		require.NoError(t, svc.Process(t.Context(), suppressed))
	})

	t.Run("Failed event is stored with no side effect", func(t *testing.T) {
		t.Parallel()

		// email.failed is a send-side failure (quota, API key, domain config), not
		// a recipient-side failure. Subscriber health must not be touched.
		failed := engagement.EmailEvent{Type: engagement.EmailEventTypeFailed, EventID: "evt_failed", Email: "nomail@example.com"}
		events, _, svc := setup(t)
		events.EXPECT().ExistsByEventID(gomock.Any(), "evt_failed").Return(false, nil)
		events.EXPECT().Create(gomock.Any(), failed).Return(failed, nil)

		require.NoError(t, svc.Process(t.Context(), failed))
	})

	t.Run("Delivery delayed event is stored with no side effect", func(t *testing.T) {
		t.Parallel()

		delayed := engagement.EmailEvent{Type: engagement.EmailEventTypeDeliveryDelayed, EventID: "evt_delayed", Email: "slow@example.com"}
		events, _, svc := setup(t)
		events.EXPECT().ExistsByEventID(gomock.Any(), "evt_delayed").Return(false, nil)
		events.EXPECT().Create(gomock.Any(), delayed).Return(delayed, nil)

		require.NoError(t, svc.Process(t.Context(), delayed))
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

	newSvc := func(t *testing.T, finder stubItemFinder) (*mockengagement.MockEmailEventRepository, *engagementsvc.EventService) {
		t.Helper()
		ctrl := gomock.NewController(t)
		events := mockengagement.NewMockEmailEventRepository(ctrl)
		subs := mockaudience.NewMockSubscriberService(ctrl)
		return events, engagementsvc.NewEvents(events, subs, finder, "admin@example.com")
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
