// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package logutil configures structured logging so errors stay readable.
//
// By default slog formats error attributes with %+v, which for errors built
// with github.com/pkg/errors prints the message followed by a full stack
// trace for every Wrap in the chain, all on the same log line. On Vercel
// that line can grow large enough to be cut short, losing the very message
// that explains the failure. The Handler in this package writes each error
// as its message alone, and writes the root cause's stack trace as a
// separate record directly after it.
package logutil

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"strings"

	"github.com/pkg/errors"
)

// StackKey is the attribute key that holds the stack trace on the record
// written after a record that logged an error carrying one.
const StackKey = "stack"

// Handler is a slog.Handler that renders error attributes as their message
// and logs the root cause's stack trace, if any, as its own record.
type Handler struct {
	next slog.Handler
}

var _ slog.Handler = (*Handler)(nil)

// NewHandler returns a Handler that writes through next.
func NewHandler(next slog.Handler) *Handler {
	return &Handler{next: next}
}

// Install wraps the current default slog handler with a Handler and makes it
// the default, so log output keeps its existing format. It is safe to call
// more than once; later calls are no-ops.
//
// slog.SetDefault redirects the standard log package into the new handler.
// slog's built-in default handler writes through the log package, so that
// redirect would loop back into itself and deadlock. Install restores the
// log package's original writer and flags afterwards, which breaks the
// cycle and leaves log.Print output exactly as it was.
func Install() {
	current := slog.Default().Handler()
	if _, ok := current.(*Handler); ok {
		return
	}
	w, flags := log.Writer(), log.Flags()
	slog.SetDefault(slog.New(NewHandler(current)))
	log.SetOutput(w)
	log.SetFlags(flags)
}

// Enabled reports whether the wrapped handler handles records at level.
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

// Handle writes r with every error attribute replaced by its message. When
// one of those errors carries a stack trace, a second record with the same
// level and message plus a " (stack trace)" suffix is written holding the
// trace of the deepest error in the chain under StackKey.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	out := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)

	var stack string
	r.Attrs(func(a slog.Attr) bool {
		replaced, s := replaceErrors(a)
		if stack == "" {
			stack = s
		}
		out.AddAttrs(replaced)
		return true
	})

	if err := h.next.Handle(ctx, out); err != nil {
		return err
	}
	if stack == "" {
		return nil
	}

	trace := slog.NewRecord(r.Time, r.Level, r.Message+" (stack trace)", r.PC)
	trace.AddAttrs(slog.String(StackKey, stack))
	return h.next.Handle(ctx, trace)
}

// WithAttrs returns a Handler whose wrapped handler has attrs added, with
// any error attributes replaced by their message.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	replaced := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		replaced[i], _ = replaceErrors(a)
	}
	return NewHandler(h.next.WithAttrs(replaced))
}

// WithGroup returns a Handler whose wrapped handler opens the group name.
func (h *Handler) WithGroup(name string) slog.Handler {
	return NewHandler(h.next.WithGroup(name))
}

// replaceErrors returns a with any error values, including those nested in
// groups, replaced by their message. It also returns the stack trace of the
// first error found that carries one, or "" if none does.
func replaceErrors(a slog.Attr) (slog.Attr, string) {
	a.Value = a.Value.Resolve()

	switch a.Value.Kind() {
	case slog.KindGroup:
		var stack string
		group := a.Value.Group()
		replaced := make([]slog.Attr, len(group))
		for i, ga := range group {
			var s string
			replaced[i], s = replaceErrors(ga)
			if stack == "" {
				stack = s
			}
		}
		return slog.Attr{Key: a.Key, Value: slog.GroupValue(replaced...)}, stack
	case slog.KindAny:
		err, ok := a.Value.Any().(error)
		if !ok || err == nil {
			return a, ""
		}
		return slog.String(a.Key, err.Error()), rootStack(err)
	default:
		return a, ""
	}
}

// stackTracer is implemented by errors from github.com/pkg/errors that
// recorded the call stack where they were created or wrapped.
type stackTracer interface {
	StackTrace() errors.StackTrace
}

// rootStack returns the stack trace of the deepest error in err's chain that
// carries one, which points closest to where the failure began. It returns
// "" if no error in the chain carries a stack trace.
func rootStack(err error) string {
	var deepest stackTracer
	for e := err; e != nil; e = errors.Unwrap(e) {
		if st, ok := e.(stackTracer); ok {
			deepest = st
		}
	}
	if deepest == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%+v", deepest.StackTrace()))
}
