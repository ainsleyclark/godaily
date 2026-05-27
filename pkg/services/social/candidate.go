// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/candidate"
)

// candidateByKind returns the first registered candidate whose Kind
// matches, or nil if none match.
func candidateByKind(all []candidate.Candidate, kind social.PostKind) candidate.Candidate {
	for _, c := range all {
		if c.Kind() == kind {
			return c
		}
	}
	return nil
}
