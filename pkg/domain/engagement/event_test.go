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
