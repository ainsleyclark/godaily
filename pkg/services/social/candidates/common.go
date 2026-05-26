// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package candidates

// platformAnchor is the platform used as the "have I covered this already?"
// probe. Any consistently-configured platform works; bluesky is always wired
// up in practice. The actual post still goes to every configured platform via
// the publish loop.
const platformAnchor = "bluesky"
