// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ingest

import (
	"fmt"
	"net/url"
	"time"
)

const scraperAPIBase = "http://api.scraperapi.com"

// ScraperURL proxies targetURL through ScraperAPI using the key selected by
// day-of-month modulo, so each key is used on alternating days. Returns
// targetURL unchanged when keys is empty.
func ScraperURL(keys []string, targetURL string) string {
	if len(keys) == 0 {
		return targetURL
	}
	key := keys[time.Now().UTC().Day()%len(keys)]
	return fmt.Sprintf("%s?api_key=%s&url=%s", scraperAPIBase, key, url.QueryEscape(targetURL))
}
