# Social Media Auto-Posting

Automatically publish the daily AI-synthesised post to social platforms immediately after the digest is sent.

## Overview

The `godaily synth` command already produces a 280-character, professionally toned post. This feature pipes that output directly to one or more social platforms without manual intervention.

## Platforms

| Platform | Priority | Notes |
|----------|----------|-------|
| Bluesky | High | Active Go community; simple app-password auth |
| LinkedIn | High | Professional reach; OAuth2 token |
| Mastodon | Medium | Already a news source; easy API |
| Twitter/X | Low | Paid API tier required |

## Shared Interface

Each platform adapter implements a single-method interface:

```go
type Poster interface {
    Post(ctx context.Context, text string) error
}
```

This keeps the orchestration layer platform-agnostic — the `run` command iterates over whichever adapters are configured and calls `Post`.

## Configuration

Each platform is opt-in via environment variables. If a platform's credentials are absent, it is silently skipped.

| Variable | Platform |
|----------|----------|
| `BLUESKY_HANDLE` + `BLUESKY_APP_PASSWORD` | Bluesky |
| `LINKEDIN_ACCESS_TOKEN` | LinkedIn |
| `MASTODON_INSTANCE` + `MASTODON_ACCESS_TOKEN` | Mastodon |
| `TWITTER_BEARER_TOKEN` | Twitter/X |

## Workflow Integration

Posting happens as a final step in the existing GitHub Actions cron workflow, after `godaily send` confirms a successful delivery.
