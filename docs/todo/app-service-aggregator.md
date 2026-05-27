# Replace App's flat service fields with the Service aggregator

## Problem

`pkg/app.go` currently exposes every domain service as a top-level field on
`App`:

```go
type App struct {
    Config         *env.Config
    DB             *sql.DB
    Repository     *Repository
    Runner         digest.Service
    Social         social.Service
    Cache          cache.Store
    Subscribers    audience.SubscriberService
    EmailEvents    engagement.EventService
    Slack          slack.Sender
    MetricsService engagement.MetricsService
    StatFetchers   map[social.Platform]platform.StatFetcher
}
```

Handlers and CLI commands reach into the App for whichever service they need
(`a.Social`, `a.MetricsService`, `a.EmailEvents`, `a.Runner`, …). The App
struct is doing two jobs: it is both the bootstrap container for
infrastructure (DB, config, cache, slack) **and** the directory of domain
services, with no separation between them.

The same file already declares a `Service` aggregator type intended to hold
just the domain services:

```go
type Service struct {
    Digest      digest.Service
    Subscribers audience.SubscriberService
    Social      social.Service
    Metrics     engagement.MetricsService
    Events      engagement.EventService
}
```

…but nothing populates or consumes it. Every call site still goes through
the flat App fields.

## Why this needs picking up

- Adding a new domain service today means adding a new top-level field on
  App, which keeps the struct growing and blurs the line between
  "infrastructure the app owns" and "domain operations the app exposes".
- Handlers receive `*godaily.App` and pick out whatever they want, so the
  dependency surface of each handler is invisible from its constructor.
- The Service aggregator already exists and signals the intended direction,
  but is dead code until something consumes it.

The next agent should rationalise this: domain services live on the
aggregator, infrastructure stays on App, and call sites consume the
aggregator instead of reaching into App directly.
