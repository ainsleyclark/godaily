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

package subscribers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"strings"

	"github.com/ainsleyclark/godaily/internal/store"
	"github.com/pkg/errors"
)

// tokenBytes is the entropy size for confirm/unsubscribe tokens. 32 bytes
// becomes 43 base64url characters — enough to make brute-force discovery
// infeasible while still fitting comfortably in a URL.
const tokenBytes = 32

// Subscribe inserts a new subscriber row with freshly generated confirm
// and unsubscribe tokens. The email is lowercased and trimmed; an empty
// email returns an error without touching the DB.
func (s *store.Store) Subscribe(ctx context.Context, email string) (store.Subscriber, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return store.Subscriber{}, errors.New("email is required")
	}

	confirm, err := newToken()
	if err != nil {
		return store.Subscriber{}, errors.Wrap(err, "generating confirm token")
	}
	unsubscribe, err := newToken()
	if err != nil {
		return store.Subscriber{}, errors.Wrap(err, "generating unsubscribe token")
	}

	sub, err := s.CreateSubscriber(ctx, CreateSubscriberParams{
		Email:            email,
		ConfirmToken:     confirm,
		UnsubscribeToken: unsubscribe,
	})
	if err != nil {
		return store.Subscriber{}, errors.Wrap(err, "creating subscriber")
	}
	return sub, nil
}

func newToken() (string, error) {
	buf := make([]byte, tokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
