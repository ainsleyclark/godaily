// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ingest

import (
	"net/url"
	"time"
)

const scraperAPIBase = "http://api.scraperapi.com"

// ScraperOption mutates the query parameters of a ScraperAPI proxy URL.
type ScraperOption func(url.Values)

// WithKeepHeaders adds keep_headers=true so ScraperAPI forwards the caller's
// request headers through to the target host. Without it ScraperAPI strips all
// client headers, so any User-Agent/Cookie set on the request never reaches the
// origin. Sources that rely on browser-like headers (e.g. Reddit's .json
// endpoint, which 403s bare datacenter requests) must enable this and also send
// those headers on the Fetch request itself.
func WithKeepHeaders() ScraperOption {
	return func(v url.Values) { v.Set("keep_headers", "true") }
}

// ScraperURL proxies targetURL through ScraperAPI using the key selected by
// day-of-month modulo, so each key is used on alternating days. Returns
// targetURL unchanged when keys is empty.
//
// premium=true routes through ScraperAPI's residential proxy pool, which is
// required for hosts that block datacenter IPs (e.g. reddit.com returns 403 on
// standard proxies).
func ScraperURL(keys []string, targetURL string, opts ...ScraperOption) string {
	if len(keys) == 0 {
		return targetURL
	}
	key := keys[time.Now().UTC().Day()%len(keys)]
	params := url.Values{
		"api_key": {key},
		"premium": {"true"},
		"url":     {targetURL},
	}
	for _, opt := range opts {
		opt(params)
	}
	return scraperAPIBase + "?" + params.Encode()
}
