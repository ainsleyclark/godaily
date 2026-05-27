// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package prompts is the parent package for social-post prompt builders.
// It owns the canonical brand language file (brand.md) shared by the
// featured and rotation sub-packages.
package prompts

import _ "embed"

// BrandRules is the canonical voice and style guide for every social post
// GoDaily publishes. Both the featured and rotation prompt builders embed
// this string into their system prompts so every generated post follows
// the same brand language.
//
//go:embed brand.md
var BrandRules string
