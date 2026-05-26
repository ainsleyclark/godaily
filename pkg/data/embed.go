// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package data exposes the curated YAML files that drive both the email
// digest (conferences with notify_dates) and the social community-promo
// rotation (conferences + meetups, both with social handles).
package data

import _ "embed"

//go:embed conferences.yaml
var Conferences []byte

//go:embed conferences-watch.yaml
var ConferencesWatch []byte

//go:embed meetups.yaml
var Meetups []byte
