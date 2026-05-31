// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import "time"

// PostWithMetrics combines a social post with its latest engagement counts.
type PostWithMetrics struct {
	ID          int64     `json:"id"`
	IssueID     *int64    `json:"issue_id,omitempty"`
	Kind        PostKind  `json:"kind"`
	Subject     string    `json:"subject,omitempty"`
	Platform    string    `json:"platform"`
	Text        string    `json:"text"`
	PostURL     string    `json:"post_url,omitempty"`
	PostedAt    time.Time `json:"posted_at"`
	Likes       int64     `json:"likes"`
	Reposts     int64     `json:"reposts"`
	Comments    int64     `json:"comments"`
	Impressions int64     `json:"impressions"`
}
