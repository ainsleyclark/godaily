// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ainsleydev/webkit/pkg/webkit"
)

func TestOK(t *testing.T) {
	t.Parallel()

	t.Run("With Body", func(t *testing.T) {
		t.Parallel()

		kit := webkit.New()

		kit.Get("/ok", func(c *webkit.Context) error {
			return OK(c, http.StatusOK, 1, "Message")
		})

		rr := httptest.NewRecorder()
		kit.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/ok", nil))

		want := `{"data":1, "error":false, "message":"Message", "status":200}`
		assert.JSONEq(t, want, rr.Body.String())
	})

	t.Run("Default Body", func(t *testing.T) {
		t.Parallel()

		kit := webkit.New()

		kit.Get("/ok", func(c *webkit.Context) error {
			return OK(c, http.StatusOK, nil, "Message")
		})

		rr := httptest.NewRecorder()
		kit.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/ok", nil))

		want := `{"data":{}, "error":false, "message":"Message", "status":200}`
		assert.JSONEq(t, want, rr.Body.String())
	})
}

func TestError(t *testing.T) {
	t.Parallel()

	kit := webkit.New()

	kit.Get("/error", func(c *webkit.Context) error {
		return Error(c, http.StatusInternalServerError, "Message")
	})

	rr := httptest.NewRecorder()
	kit.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/error", nil))

	want := `{"status":500,"error":true,"message":"Message"}`
	assert.JSONEq(t, want, rr.Body.String())
}

func TestValidationErrorMessage(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		input error
		want  string
	}{
		"Nil Error": {
			input: nil,
			want:  "Invalid Request",
		},
		"With Error": {
			input: errors.New("field 'name' is required"),
			want:  "Invalid Request - field 'name' is required",
		},
		"Empty Error Message": {
			input: errors.New(""),
			want:  "Invalid Request",
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := ValidationErrorMessage(test.input)
			assert.Equal(t, test.want, got)
		})
	}
}
