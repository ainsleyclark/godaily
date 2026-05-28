# Social platform mentions

## Problem

GoDaily posts to three platforms (Bluesky, Mastodon, LinkedIn). When a
source has no platform-specific handle the post falls back to the plain
display name, so the source never sees the post and readers get no
clickable profile link.

Bluesky and Mastodon mentions are wired up and working in production.
LinkedIn organisation and person mentions are plumbed end-to-end but no
URNs are populated yet — drop them into the profiles list and they light
up.

## Data model

One value type unifies every platform's mentions:

```go
type Mention struct {
    Platform    Platform   // bluesky / mastodon / linkedin
    DisplayName string     // case-sensitive substring to find in text (LinkedIn)
    Handle      string     // @handle or urn:li:organization:<id> / urn:li:person:<id>
}
```

The same shape covers `@-handles` (Bluesky, Mastodon — inlined directly
in post text by the prompt layer) and LinkedIn URNs (attached as
out-of-band annotations).

`Profile.Mentions` is `[]Mention`. `CandidateContext.Mentions` is
`[]Mention`. `PostRequest.Mentions` is `[]Mention`. Same name, same
shape, everywhere.

## Current handle coverage

| Source             | Bluesky                        | Mastodon                       | LinkedIn |
|--------------------|--------------------------------|--------------------------------|----------|
| Ardan Labs         | `@ardanlabs.com`               | `@ardanlabs@hachyderm.io`      | —        |
| Go Blog            | `@golang.org`                  | `@golang@hachyderm.io`         | —        |
| JetBrains          | `@jetbrains.com`               | `@jetbrains@mastodon.social`   | —        |
| DEV Community      | `@thepracticaldev.bsky.social` | `@thepracticaldev@mas.to`      | —        |
| go podcast()       | —                              | `@dmitshur@hachyderm.io`       | —        |
| Fallthrough        | `@fallthrough.fm`              | —                              | —        |
| Lobsters           | —                              | —                              | —        |
| Go Vuln            | —                              | —                              | —        |
| Awesome Go         | —                              | —                              | —        |
| Go Releases        | —                              | —                              | —        |
| GitHub Trending    | —                              | —                              | —        |
| Go Proposals       | —                              | —                              | —        |
| Go Conferences     | —                              | —                              | —        |
| Go Meetups         | —                              | —                              | —        |
| GolangBridge       | —                              | —                              | —        |
| Go talks (YouTube) | —                              | —                              | —        |

`—` means no handle is configured; the post falls back to the display name.

## LinkedIn mentions: how the wiring works

LinkedIn's versioned `/rest/posts` API (currently v202601) does not take a
separate `mentionedOrganizations` field. Mentions are *inline annotations
on the `commentary` field*: a `commentaryAnnotations` array of
`{start, length, entity}` tuples where `entity` is the
`urn:li:organization:<id>` or `urn:li:person:<id>` URN and the
(start, length) range points at the matching span of `commentary`.

Two constraints matter:

1. The visible text in `commentary` must match the entity's name on
   LinkedIn **case-sensitively**. If it doesn't, LinkedIn renders the
   text as plain text.
2. LinkedIn rolls API versions ~every quarter and renames fields between
   them. Pin `LINKEDIN_API_VERSION` in env config and re-verify the
   annotation shape against the live docs before relying on it.

### Multiple mentions per post

`buildAnnotations` in `pkg/services/social/platform/linkedin/linkedin.go`
takes the full `[]Mention` slice, finds the first case-sensitive
occurrence of each `DisplayName` in the post text, and produces one
inline annotation per match. Overlap resolution: if two mentions could
match overlapping ranges (e.g. "Go" inside "Go Blog"), the **longer
match wins**. Ties go to the earlier position. Each URN is annotated at
most once per post.

### Missed-mention telemetry

When an intended mention's `DisplayName` doesn't appear in the post
text, the LinkedIn poster logs a `WARN` line:

```
LinkedIn mention dropped: display name not found in post text
  display_name=<X> handle=<urn:...>
```

The post still goes through, just without a tag for that entity. Watch
these logs after rolling out a new prompt — if the LLM has stopped
including a name reliably, you'll see it here. (Surfacing this through
to Slack is a follow-up.)

### Adding a LinkedIn URN to an existing profile

Edit the entry in `pkg/domain/social/profile.go` and append `Mention`
entries to its `Mentions` slice. For the company:

```go
news.SourceArdanLabs: {
    Source:      news.SourceArdanLabs,
    DisplayName: "Ardan Labs",
    Mentions: []social.Mention{
        {Platform: social.Bluesky,  Handle: "@ardanlabs.com"},
        {Platform: social.Mastodon, Handle: "@ardanlabs@hachyderm.io"},
        {Platform: social.LinkedIn, DisplayName: "Ardan Labs",      Handle: "urn:li:organization:1337"},
        {Platform: social.LinkedIn, DisplayName: "William Kennedy", Handle: "urn:li:person:42"},
        // Alias — short-form name maps to same person URN. Both spellings
        // are matched independently; whichever the LLM happens to use
        // gets annotated.
        {Platform: social.LinkedIn, DisplayName: "Bill Kennedy",    Handle: "urn:li:person:42"},
    },
    // ...
},
```

The numeric organisation id is visible to page admins in the URL bar on
`linkedin.com/company/<slug>/admin/`, or via the LinkedIn
`/organizationAcls` API. Person URNs come from the People Typeahead
API or by inspecting the profile URL on a logged-in session.

**Make sure each LinkedIn `DisplayName` case-sensitively appears in the
rendered post copy** — otherwise the annotation is silently dropped and
you'll see a WARN line in the logs.

## Per-article author mentions (not yet)

Currently `Profile` is one-per-source — there's no slot for "tag the
author of *this specific article*." For featured posts that mention the
article's source you get the source's full mention list, which works
when the source is small enough that its standard author always applies
(Ardan Labs → Bill Kennedy). For sources with rotating authors (Go Blog,
JetBrains) the per-source approach won't pick the right person.

The expensive fix is a new model: `news.Item.Author` populated by the
ingest layer (RSS `<author>`, Atom `<name>`, GitHub commit author), plus
a people directory or per-source author overrides. Defer until the
per-source approach is exhausted.

## Outstanding handle gaps

Research needed before adding:

- **go podcast() on Bluesky** — Dmitri Shuralyov's Bluesky handle.
- **Fallthrough on Mastodon** — confirm the show has a Mastodon account.
- **Lobsters, Awesome Go, Go Vuln** — may or may not have canonical
  social accounts.

## Aggregated sources (no handle expected)

These are aggregated feeds with no single creator account to tag.
Falling back to the display name is the correct behaviour:

- GitHub Trending (Go)
- Go Conferences
- Go Meetups
- GolangBridge
- Go talks on YouTube
- Go Proposals (tracker)
