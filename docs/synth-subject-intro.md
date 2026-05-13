



## 1. Daily TL;DR intro for the email

A 2–3 sentence "what mattered today" paragraph rendered at the top of
the digest email, generated from the same item list already passed to
`Suggest`.

- **Why it's high ROI:** reuses the cached system block (cheap on
  cache-read), one new field on `Suggestion`, one new slot in the
  email template.
- **Where it lands:** new `TLDR string` on `synth.Suggestion`,
  populated from the same model call (extend the JSON schema in
  `systemIntro`), rendered in `internal/cron/email.html` and
  `email.txt`.
- **Tradeoff:** widens the prompt's output contract slightly. If the
  model fails the TL;DR but produces good posts, do we still ship?
  Probably yes — keep TL;DR optional in the parse step.

## 4. Email subject line generation

The digest currently uses a static subject. A one-line teaser drawn
from the top item would lift open rates and is essentially free given
the synth call already happens.

- **Where it lands:** another field on `Suggestion`, threaded through
  to `email.SendEmailRequest.Subject` in `internal/cron/email.go`.
