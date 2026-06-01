// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplayContentType(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"examples/raw/golang_cafe.html": "text/html",
		"page.htm":                      "text/html",
		"we_work_remotely.xml":          "application/xml",
		"feed.rss":                      "application/xml",
		"feed.atom":                     "application/xml",
		"remotive.json":                 "application/json",
		"capture.txt":                   "text/plain",
		"noextension":                   "text/plain",
	}
	for path, want := range cases {
		assert.Equal(t, want, replayContentType(path), path)
	}
}

func TestStaticTransport(t *testing.T) {
	t.Parallel()
	body := []byte("captured-response-body")
	client := &http.Client{Transport: &staticTransport{body: body, contentType: "application/xml"}}

	// Any URL yields the captured body — this is how -replay feeds a source's
	// Fetch the past-day capture regardless of the source's internal URL.
	resp, err := client.Get("https://example.com/anything")
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/xml", resp.Header.Get("Content-Type"))
	got, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, body, got)

	// Follow-up enrichment calls receive the same bytes rather than erroring.
	resp2, err := client.Get("https://example.com/enrich")
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp2.Body.Close() })
	got2, err := io.ReadAll(resp2.Body)
	require.NoError(t, err)
	assert.Equal(t, body, got2)
}
