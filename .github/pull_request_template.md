## What and why

<!-- One or two sentences. Link the issue: "Closes #123" or "Refs #123". -->

## Changes

-

## Testing

- [ ] `go test ./...`
- [ ] `golangci-lint run ./... --fix --config=.golangci.yaml`

## Verification

<!--
How we'll know this worked in PRODUCTION, not just in tests. The nightly
autopilot (.claude/skills/autopilot) reads this block to verify the fix in a
later run, so fill in every line. Write "n/a" only for changes with no
runtime effect (docs, tooling).
-->

- **Signal:** <!-- what changes when it works, e.g. "featured posts end with a Subscribe: line" -->
- **Where to look:** <!-- MCP tool / Vercel log query / API call, e.g. "social_post_metrics from=<date>" -->
- **First observable:** <!-- the first date the code actually runs after deploy, e.g. "Fri 27 Nov, posts drafted at 02:00 UTC" -->
- **Pass if:** <!-- a concrete, checkable condition -->
- **Fail if:** <!-- what would show it didn't work -->
