// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"fmt"

	"github.com/ainsleydev/webkit/pkg/webkit"
)

type (
	// Response represents the data sent back from the API.
	// Indicating if there was an error processing the requests, a
	// status code, a message and the data that was processed.
	Response struct {
		Status  int    `json:"status"`
		Error   bool   `json:"error"`
		Message string `json:"message" example:"User formatted message from the API"`
		Data    any    `json:"data,omitempty" swaggertype:"object"`
	} //@name Response
)

const (
	// ErrDecodeBodyMessage is returned by a handler when a
	// request body could not be unmarshalled.
	ErrDecodeBodyMessage = "Error decoding response body"

	// ErrInvalidID is returned by a handler when an
	// ID or path value could not be parsed.
	ErrInvalidID = "Error invalid ID"
)

// OK sends a successful response with the given HTTP
// status code, data, and message, wrapping the
// response in the standard API Response struct.
func OK(ctx *webkit.Context, status int, data any, message string) error {
	if data == nil {
		data = make(map[string]any) // Not null
	}
	return ctx.JSON(status, Response{
		Status:  status,
		Error:   false,
		Message: message,
		Data:    data,
	})
}

// Error sends an error response with the given HTTP
// status code, data, and message, wrapping the
// response in the standard API Response struct.
func Error(ctx *webkit.Context, status int, message string) error {
	return ctx.JSON(status, Response{
		Status:  status,
		Error:   true,
		Message: message,
		Data:    nil,
	})
}

// ValidationErrorMessage returns a formatted validation
// error message for when validation failed.
func ValidationErrorMessage(err error) string {
	const msg = "Invalid Request"
	if err == nil {
		return msg
	}
	if err.Error() == "" {
		return msg
	}
	return fmt.Sprintf("%s - %s", msg, err.Error())
}
