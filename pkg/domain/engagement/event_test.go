// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package engagement_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
)

func TestEmailEventType_Valid(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		input engagement.EmailEventType
		want  bool
	}{
		"Delivered":  {input: engagement.EmailEventTypeDelivered, want: true},
		"Opened":     {input: engagement.EmailEventTypeOpened, want: true},
		"Clicked":    {input: engagement.EmailEventTypeClicked, want: true},
		"Bounced":    {input: engagement.EmailEventTypeBounced, want: true},
		"Complained": {input: engagement.EmailEventTypeComplained, want: true},
		"Unknown":    {input: engagement.EmailEventType("exploded"), want: false},
		"Empty":      {input: engagement.EmailEventType(""), want: false},
		"Wire form":  {input: engagement.EmailEventType("email.opened"), want: false},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, test.input.Valid())
		})
	}
}

func TestEmailEventType_String(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "clicked", engagement.EmailEventTypeClicked.String())
}
