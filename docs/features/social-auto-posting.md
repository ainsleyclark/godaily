# Social Media Auto-Posting

Automatically publish the daily AI posts to social platforms immediately after the digest is sent.

## Overview

In order to get more engagement, I want automatic GoDaily posts to post to the following social 
platforms, containing an engaging social post, tailored to each platform.

## Platforms

I have already signed up to these platforms and have profiles and the correct api keys in order to
start.

| Platform | Priority | Notes                                         |
|----------|----------|-----------------------------------------------|
| Bluesky  | High     | Active Go community; simple app-password auth |
| LinkedIn | High     | Professional reach; OAuth2 token              |
| Mastodon | Medium   | Already a news source; easy API               |

## Shared Interface

Each platform adapter implements a single-method interface:

```go
type Poster interface {
Post(ctx context.Context, text string) error
}
```

This keeps the orchestration layer platform-agnostic.

## Configuration

Each platform is opt-in via environment variables. If a platform's credentials are absent, it is
skipped. We may need to add more env vars that define handles, but if it can be hard coded that's
fine as it's not sensitive.

| Variable               | Platform |
|------------------------|----------|
| `BLUESKY_APP_PASSWORD` | Bluesky  |
| `LINKEDIN_OAUTH_TOKEN` | LinkedIn |
| `MASTODON_APP_TOKEN`   | Mastodon |

## Prompts

Each social platform should have its own prompt, tailored for that specific platform to boost
engagement, I will take your steer on this. But each platform should probably call ai.Prompter with
a prompt to get the content from the Issue.

## Content

- We should obtain all of the news items from the database and use it as a base to formulate a nice
  social post using a prompt.
- We should vary content, sometimes it should be a video, sometimes a proposal, but I want to
  priortise the most useful content such as proposals and rich articles.
- We should always use the most relevant hashtags, perhaps these can be defined as variablesv for
  easy editing.
- Sometimes, posts shouldn't happen, or skip days, as it may appear that it's being posted by a
  person. But if you think this should be removed or tweaked let me know

## Implementation

- This code should live under a package called social, under social is where we would keep all of
  the providers, for example, `social/linkedin`.
- If possible, we should use a GoLang SDK's for each social provider, but if that's not possible,
  just use stdlib.
- We shoudld create a new mock for the `Poster`.
- It would be good to have a CLI command in order to invoke this for testing, with a dry run mode.
- Perhaps we need a new social service under social?

## Workflow Integration

Posting should happen as a separate cron job, a separate api route called `/social` that should run
every morning at around 11.30am. However, it might be great to randomise this time so it looks like
it's being posted from a human.

Make sure you read AGENTS.md and ask me any questions if you need to.


MOST IMPORTANTLY, we need engagement, that needs to be prioritised.
