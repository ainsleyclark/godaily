// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/candidate"
)

// Re-exports of the candidate package types so existing rotation code in
// this package keeps reading naturally. The interface itself lives in
// pkg/services/social/candidate to break the import cycle between this
// package and pkg/services/social/candidates.
type (
	Candidate        = candidate.Candidate
	CandidateContext = candidate.CandidateContext
	Generator        = candidate.Generator
)

func candidateByKind(all []Candidate, kind social.PostKind) Candidate {
	for _, c := range all {
		if c.Kind() == kind {
			return c
		}
	}
	return nil
}
