# AI - Subjects & Intros

It would be great if AI could generate a subject and intro for the emails. Details below.

## Daily TL;DR intro for the email

A sentence or two on "what mattered today" paragraph rendered at the top of
the digest email, generated from the same item list already passed to
`Suggest`.

- **Why it's high ROI:** reuses the cached system block (cheap on
  cache-read), one new field on `Suggestion`, one new slot in the
  email template.

## Email subject line generation

The digest currently uses a static subject. A one-line teaser drawn
from the top item would lift open rates and is essentially free given
the synth call already happens.

- **Where it lands:** another field on `Suggestion`, threaded through
  to `email.SendEmailRequest.Subject` in `internal/cron/email.go`.

This will appear in the subject but should be called "title" in the Issue, then we can use this as
the card title in the front-end where we render issues.

## Notes

- We will need to add these fields into the databaase.
- I want to know my options on how we can wire this up with the existing Synth. Should it be
  separate calls? I'm worried the synth package is getting confusing and hard to manage.


Come up with a plan, read AGENTS.md 
