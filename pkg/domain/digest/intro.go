// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import "strings"

// IntroParagraphs splits an issue intro/summary into display paragraphs on line
// breaks. The generator and the dashboard editor use blank lines to separate
// distinct subjects so the intro no longer reads as one wall of text; each
// non-empty line becomes its own paragraph. Blank lines are collapsed, so a
// single break and a double break render the same. The result is nil when the
// summary is empty or whitespace only.
func IntroParagraphs(summary string) []string {
	lines := strings.Split(strings.ReplaceAll(summary, "\r\n", "\n"), "\n")
	paras := make([]string, 0, len(lines))
	for _, line := range lines {
		if line = strings.TrimSpace(line); line != "" {
			paras = append(paras, line)
		}
	}
	if len(paras) == 0 {
		return nil
	}
	return paras
}

// IntroFlattened collapses the intro's subject line breaks into a single space
// separated string. It is used wherever the summary feeds a single-line context
// — meta descriptions, JSON-LD, RSS — where literal line breaks are wrong.
func IntroFlattened(summary string) string {
	return strings.Join(IntroParagraphs(summary), " ")
}
