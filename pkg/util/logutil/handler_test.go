// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logutil

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"log/slog"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestLogger returns a logger writing text records, without timestamps,
// through a Handler into the returned buffer.
func newTestLogger() (*slog.Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	text := slog.NewTextHandler(buf, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey && len(groups) == 0 {
				return slog.Attr{}
			}
			return a
		},
	})
	return slog.New(NewHandler(text)), buf
}

func lines(buf *bytes.Buffer) []string {
	return strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
}

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	t.Run("Error with stack is split onto its own line", func(t *testing.T) {
		t.Parallel()

		logger, buf := newTestLogger()
		err := errors.Wrap(errors.Wrap(errors.New("boom"), "listing items"), "building")

		logger.Error("Build failed", "slug", "2026-11-24", "err", err)

		got := lines(buf)
		require.Len(t, got, 2)
		assert.Equal(t, `level=ERROR msg="Build failed" slug=2026-11-24 err="building: listing items: boom"`, got[0])
		assert.True(t, strings.HasPrefix(got[1], `level=ERROR msg="Build failed (stack trace)" stack=`), got[1])
		assert.Contains(t, got[1], "TestHandler_Handle")
		assert.Contains(t, got[1], "handler_test.go")
	})

	t.Run("Only the root cause stack is logged", func(t *testing.T) {
		t.Parallel()

		logger, buf := newTestLogger()
		root := errors.New("boom")
		wrapped := errors.Wrap(errors.Wrap(root, "inner"), "outer")

		logger.Error("Failed", "err", wrapped)

		got := lines(buf)
		require.Len(t, got, 2)
		want := strings.TrimSpace(fmt.Sprintf("%+v", root.(stackTracer).StackTrace()))
		assert.Equal(t, `level=ERROR msg="Failed (stack trace)" stack=`+fmt.Sprintf("%q", want), got[1])
	})

	t.Run("Error without stack stays on one line", func(t *testing.T) {
		t.Parallel()

		logger, buf := newTestLogger()

		logger.Warn("Request failed", "err", fmt.Errorf("fetching: %w", context.DeadlineExceeded))

		got := lines(buf)
		require.Len(t, got, 1)
		assert.Equal(t, `level=WARN msg="Request failed" err="fetching: context deadline exceeded"`, got[0])
	})

	t.Run("Nil error is left alone", func(t *testing.T) {
		t.Parallel()

		logger, buf := newTestLogger()
		var err error

		logger.Info("OK", "error", err)

		assert.Equal(t, []string{`level=INFO msg=OK error=<nil>`}, lines(buf))
	})

	t.Run("Errors in groups are replaced", func(t *testing.T) {
		t.Parallel()

		logger, buf := newTestLogger()

		logger.Error("Failed", slog.Group("req", slog.String("path", "/build"), slog.Any("err", errors.New("boom"))))

		got := lines(buf)
		require.Len(t, got, 2)
		assert.Equal(t, `level=ERROR msg=Failed req.path=/build req.err=boom`, got[0])
		assert.True(t, strings.HasPrefix(got[1], `level=ERROR msg="Failed (stack trace)" stack=`), got[1])
	})

	t.Run("Records without errors pass through", func(t *testing.T) {
		t.Parallel()

		logger, buf := newTestLogger()

		logger.Info("Built draft issue", "slug", "2026-11-24", "items", 22)

		assert.Equal(t, []string{`level=INFO msg="Built draft issue" slug=2026-11-24 items=22`}, lines(buf))
	})
}

func TestHandler_WithAttrs(t *testing.T) {
	t.Parallel()

	logger, buf := newTestLogger()

	logger.With("err", errors.New("boom")).Info("Hello")

	assert.Equal(t, []string{`level=INFO msg=Hello err=boom`}, lines(buf))
}

func TestHandler_WithGroup(t *testing.T) {
	t.Parallel()

	logger, buf := newTestLogger()

	logger.WithGroup("build").Error("Failed", "err", errors.New("boom"))

	got := lines(buf)
	require.Len(t, got, 2)
	assert.Equal(t, `level=ERROR msg=Failed build.err=boom`, got[0])
	assert.True(t, strings.HasPrefix(got[1], `level=ERROR msg="Failed (stack trace)" build.stack=`), got[1])
}

func TestHandler_Enabled(t *testing.T) {
	t.Parallel()

	text := slog.NewTextHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelWarn})
	h := NewHandler(text)

	assert.False(t, h.Enabled(context.Background(), slog.LevelInfo))
	assert.True(t, h.Enabled(context.Background(), slog.LevelError))
}

// TestInstall mutates the process-wide default logger, so it must not run
// in parallel with other tests that log through it.
//
//nolint:paralleltest // Mutates global slog and log state.
func TestInstall(t *testing.T) {
	prevLogger := slog.Default()
	prevOutput := log.Writer()
	prevFlags := log.Flags()
	t.Cleanup(func() {
		slog.SetDefault(prevLogger)
		log.SetOutput(prevOutput)
		log.SetFlags(prevFlags)
	})

	buf := &bytes.Buffer{}
	log.SetOutput(buf)
	log.SetFlags(0)

	Install()
	Install() // Second call must not wrap twice.

	h, ok := slog.Default().Handler().(*Handler)
	require.True(t, ok)
	_, nested := h.next.(*Handler)
	assert.False(t, nested)

	// The log package must be left writing where it was.
	assert.Equal(t, buf, log.Writer())
	assert.Equal(t, 0, log.Flags())

	// Both calls deadlock if the log package is still redirected into slog.
	slog.Error("Build failed", "err", errors.New("boom"))
	log.Print("plain log line")

	got := lines(buf)
	require.Len(t, got, 3)
	assert.Equal(t, `ERROR Build failed err=boom`, got[0])
	assert.True(t, strings.HasPrefix(got[1], `ERROR Build failed (stack trace) stack=`), got[1])
	assert.Equal(t, `plain log line`, got[2])
}
