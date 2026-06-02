# Gmail content trimming in the digest

This doc captures why the digest email must avoid blocks of markup that repeat,
near-identically, in every issue — and what happened when we added some. Read
this before adding any new recurring, boilerplate row to the digest template.

## Problem

Subscribers on Gmail saw large parts of the digest collapsed behind Gmail's
**"show trimmed content"** toggles (the `…` pills). Entire sections were hidden,
the email looked broken, and it drove unsubscribes.

This is **not** the size-based `[Message clipped]` banner. Gmail clips messages
at ~102 KB; a representative digest renders at ~28 KB, so size is not the cause.
The `…` pills are a different feature.

## Mechanism

Gmail's "show trimmed content" hides runs of content it believes are **duplicated
across a conversation** — the same heuristic that famously collapses email
signatures, because they are byte-for-byte identical on every message. Gmail:

1. Threads messages it considers part of one conversation (same/similar subject,
   same sender), and
2. Collapses the parts of a newer message that match content already shown in an
   earlier message in that thread.

A daily newsletter is the worst case for this: the structural scaffolding
(section headers, repeated link text, footer) is identical issue to issue, so
Gmail treats it as "quoted" boilerplate and trims it. The items that genuinely
change each day (article titles) stay visible; the repeated scaffolding around
them collapses. That is exactly the visible-vs-collapsed split we saw.

References:
- <https://www.labnol.org/internet/gmail-trimming-signature/28762> (the
  signature case — the canonical example of this behaviour)
- <https://sendlayer.com/blog/what-is-inbox-clipping-and-how-to-avoid-it/>

## What triggered it

PR #257 ("Add 'Read X more' browse CTAs to digest sections") added a per-section
call-to-action to the bottom of **every** section:

```html
<div style="padding:14px 0 0;border-top:1px solid #d0e8f5;">
  <a href="{{ .BrowseURL }}" ...>Read {{ .More }} more {{ .Title }} on GoDaily &rarr;</a>
</div>
```

Six structurally-identical "divider + short link ending in *on GoDaily →*" blocks
is a strong dose of repeated, near-identical boilerplate. The section headers,
`Read on X →` links, and footer already repeated every issue without obvious
trouble; the extra CTA blocks pushed Gmail's trimmer over the edge and it began
collapsing whole sections.

PR #257's email rendering was reverted to the known-good state to resolve the
incident.

## Guidance for future changes

When touching `pkg/templates/email.html` / `email.txt`:

- **Avoid adding blocks that are identical in every issue.** The more repeated,
  boilerplate markup the digest carries, the more Gmail has to collapse. New
  recurring rows are the highest-risk change.
- **Prefer one block over many.** A single footer link (e.g. "Browse all topics
  on GoDaily →") carries far less repeated weight than one CTA per section. One
  non-repeated block is unlikely to trip the trimmer; six identical ones will.
- **Content that changes per issue is safe.** Article titles, snippets, and the
  AI-generated subject/intro vary daily, so Gmail keeps them visible.
- **The only real test is a live send.** Gmail's trimmer cannot be reproduced
  offline. After changing the template, send yourself two consecutive issues and
  confirm the second one renders fully (the trimmer only acts once there is a
  prior message in the thread to diff against).

### If the trimmer ever returns

The documented mitigations, in order of preference:

1. **Reduce repeated boilerplate** — the direct lever, and what the #257 revert
   did.
2. **Keep issues in separate conversations** so Gmail has nothing to diff
   against. We set no `References` / `In-Reply-To` threading headers (see
   `buildEmailRequest` in `pkg/services/digest/email.go`), and subjects are
   AI-generated per issue, so issues should not thread — but Gmail normalises
   subjects aggressively, so this is not guaranteed.
3. **Make a repeated block unique per send** (the signature workaround: append a
   hidden per-issue token) as a last resort if a recurring block is genuinely
   required.
