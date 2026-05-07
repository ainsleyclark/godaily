// Package hook provides fire-and-forget HTTP helpers used by serverless handlers.
package hook

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
)

// Heartbeat pings a BetterStack (or compatible) heartbeat URL to signal that a
// job completed successfully. It is a no-op when url is empty.
func Heartbeat(ctx context.Context, url string) {
	fire(ctx, http.MethodGet, url, "heartbeat")
}

// Deploy triggers a Vercel deploy hook via POST. It is a no-op when url is empty.
func Deploy(ctx context.Context, url string) {
	fire(ctx, http.MethodPost, url, "deploy hook")
}

func fire(ctx context.Context, method, url, label string) {
	if url == "" {
		slog.DebugContext(ctx, "Skipping "+label+" — URL not configured")
		return
	}
	req, err := http.NewRequestWithContext(ctx, method, url, strings.NewReader(""))
	if err != nil {
		slog.ErrorContext(ctx, "Creating "+label+" request", "error", err)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.ErrorContext(ctx, "Firing "+label, "error", err)
		return
	}
	if err = resp.Body.Close(); err != nil {
		slog.ErrorContext(ctx, "Closing "+label+" response body", "error", err)
	}
}
