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

package email_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ainsleyclark/godaily/pkg/domain/email"
)

func TestEventType_Valid(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		input email.EventType
		want  bool
	}{
		"Delivered":  {input: email.EventTypeDelivered, want: true},
		"Opened":     {input: email.EventTypeOpened, want: true},
		"Clicked":    {input: email.EventTypeClicked, want: true},
		"Bounced":    {input: email.EventTypeBounced, want: true},
		"Complained": {input: email.EventTypeComplained, want: true},
		"Unknown":    {input: email.EventType("exploded"), want: false},
		"Empty":      {input: email.EventType(""), want: false},
		"Wire form":  {input: email.EventType("email.opened"), want: false},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, test.input.Valid())
		})
	}
}

func TestEventType_String(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "clicked", email.EventTypeClicked.String())
}
