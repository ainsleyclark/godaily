// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ainsleyclark/godaily/pkg/env"
	mockai "github.com/ainsleyclark/godaily/pkg/mocks/ai"
	mockdigest "github.com/ainsleyclark/godaily/pkg/mocks/digest"
	mocknews "github.com/ainsleyclark/godaily/pkg/mocks/news"
	"github.com/ainsleyclark/godaily/pkg/mocks/slack"
	mocksocial "github.com/ainsleyclark/godaily/pkg/mocks/social"
	socialsvc "github.com/ainsleyclark/godaily/pkg/services/social"
)

// newHandlerNoPosters builds a Handler with a real social.Service that
// has no posters configured. Both publish handlers (featured and
// rotation) share this fixture for their weekend / not-wired
// short-circuit assertions.
func newHandlerNoPosters(t *testing.T) *Handler {
	t.Helper()

	ctrl := gomock.NewController(t)

	slackMock := mockslack.NewMockSender(ctrl)
	slackMock.EXPECT().MustSend(gomock.Any(), gomock.Any()).AnyTimes()

	prompter := mockai.NewMockPrompter(ctrl)
	issues := mockdigest.NewMockIssueRepository(ctrl)
	items := mocknews.NewMockItemRepository(ctrl)
	posts := mocksocial.NewMockPostRepository(ctrl)

	svc, err := socialsvc.New(env.Config{}, prompter, issues, items, posts, nil, slackMock)
	require.NoError(t, err)

	return &Handler{
		social: svc,
		slack:  slackMock,
		config: &env.Config{},
	}
}
