package handlers

import (
	"fmt"
	"net/http"
	"time"
)

// ReloadHTTP is a raw http.HandlerFunc (registered on chi.Mux directly, not via
// webkit.Kit) so the response writer keeps its native http.Flusher and the SSE
// stream isn't buffered.
//
// First connect (no Last-Event-ID header) sends only a comment line so
// onmessage doesn't fire. After air restarts the binary the browser auto-
// reconnects with Last-Event-ID set; the new server replies with `data: reload`
// to trigger location.reload().
func ReloadHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")

	id := time.Now().UnixNano()
	if r.Header.Get("Last-Event-ID") != "" {
		fmt.Fprintf(w, "id: %d\ndata: reload\n\n", id)
	} else {
		fmt.Fprintf(w, "id: %d\n: hello\n\n", id)
	}
	flusher.Flush()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
