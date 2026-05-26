// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import "embed"

// Assets holds embedded fonts and images used by sub-packages such as web/og.
//
//go:embed assets/fonts/*.ttf assets/images/og-home.png assets/images/og-issue.png
var Assets embed.FS
