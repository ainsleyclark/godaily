// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package templates

import _ "embed"

//go:embed email_layout.html
var EmailLayout string

//go:embed email_layout.txt
var EmailLayoutText string

//go:embed email.html
var EmailHTML string

//go:embed email.txt
var EmailText string

//go:embed confirm.html
var ConfirmHTML string

//go:embed confirm.txt
var ConfirmText string

//go:embed suggest.html
var SuggestHTML string

//go:embed suggest.txt
var SuggestText string

//go:embed message.html
var MessageHTML string
