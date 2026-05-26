// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package rotation builds AI prompts for the Tue/Wed/Fri rotation slot —
// the non-featured social posts. Each post kind (CTA, NewSource, Recap,
// Spotlight, Community) has its own generator file; they all share the
// platformProfile rules and run() helper defined in rotation.go.
package rotation
