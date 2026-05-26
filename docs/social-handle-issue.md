# Social platform handle gaps

## Problem

GoDaily posts to three platforms (Bluesky, Mastodon, LinkedIn) but source handles
are only partially populated. This causes some posts to fall back to a plain
display name rather than a real @mention, meaning the source never sees the post
and readers get no clickable profile link.

## Current state

Handle coverage per source:

| Source            | Bluesky                        | Mastodon                       | LinkedIn |
|-------------------|-------------------------------|-------------------------------|----------|
| Ardan Labs        | `@ardanlabs.com`               | `@ardanlabs@hachyderm.io`      | —        |
| Go Blog           | `@golang.org`                  | `@golang@hachyderm.io`         | —        |
| JetBrains         | `@jetbrains.com`               | `@jetbrains@mastodon.social`   | —        |
| DEV Community     | `@thepracticaldev.bsky.social` | `@thepracticaldev@mas.to`      | —        |
| go podcast()      | —                              | `@dmitshur@hachyderm.io`       | —        |
| Fallthrough       | `@fallthrough.fm`              | —                              | —        |
| Lobsters          | —                              | —                              | —        |
| Go Vuln           | —                              | —                              | —        |
| Awesome Go        | —                              | —                              | —        |
| Go Releases       | —                              | —                              | —        |
| GitHub Trending   | —                              | —                              | —        |
| Go Proposals      | —                              | —                              | —        |
| Go Conferences    | —                              | —                              | —        |
| Go Meetups        | —                              | —                              | —        |
| GolangBridge      | —                              | —                              | —        |
| Go talks (YouTube)| —                              | —                              | —        |

`—` means no handle is configured; the post falls back to the display name.

## LinkedIn is a structural limitation

LinkedIn's `/rest/posts` API does not support text-level @mentions. Mentioning
an organisation requires:

1. The organisation's numeric URN (`urn:li:organization:<id>`).
2. A separate `mentionedOrganizations` (or equivalent) field in the API request
   body — the exact field name varies by API version and must be confirmed
   against the LinkedIn API v202601 docs.
3. The post text itself still uses the plain company name; LinkedIn substitutes
   the linked mention in its client.

This requires changes to:
- `pkg/domain/social/profile.go` — add a `LinkedInURN string` field
- `pkg/services/social/platform/platform.go` — change `Post(ctx, text)` to
  `Post(ctx, PostRequest)` so LinkedIn URNs can be threaded through without
  changing other platforms
- `pkg/services/social/platform/linkedin/linkedin.go` — populate
  `mentionedOrganizations` in the request body when a URN is present
- `pkg/services/social/service.go` — pass the URN from profile through to
  `publish()`

LinkedIn org IDs for known sources need to be looked up (the numeric ID appears
in the source URL on linkedin.com/company/<slug> or via the LinkedIn API).

## Missing Bluesky/Mastodon handles

Some sources with real social accounts are not yet wired up:

- **go podcast() on Bluesky**: Dmitri Shuralyov's Bluesky handle needs to be
  confirmed and added to `profile.go`.
- **Fallthrough on Mastodon**: The show's Mastodon handle (if one exists) needs
  to be confirmed and added.
- **Lobsters, Awesome Go, Go Vuln, etc.**: These sources may or may not have
  active Bluesky/Mastodon accounts. Research needed before adding handles.

## Aggregated sources (no handle expected)

The following sources are aggregated feeds with no single creator account to tag.
Falling back to the display name is the correct behaviour for these:

- GitHub Trending (Go)
- Go Conferences
- Go Meetups
- GolangBridge
- Go talks on YouTube
- Go Proposals (tracker)
