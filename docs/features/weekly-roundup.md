# Weekly AI Roundup

Produce a longer-form weekly summary of the week's top Go news, published every Friday, to
complement the daily digest and create more shareable, SEO-friendly content.

## Overview

The daily digest is optimised for quick scanning. A weekly roundup serves a different need: a richer
narrative that's worth bookmarking, sharing on LinkedIn, and indexing by search engines.

## Content

Claude synthesises the top items across all five issues from the week into a 3–5 paragraph roundup
covering:

- The single most significant story
- Releases and proposals that shipped
- Community discussion highlights
- Recommended reading for the weekend

## Distribution

1. Published as a web page at `/weekly/:year/:week`
2. Emailed to subscribers (opt-in toggle, or same list)
3. Auto-posted to LinkedIn and Bluesky via the social auto-posting adapter

## Automation

A new `godaily roundup` CLI command runs on Friday via the existing GitHub Actions cron
infrastructure. It reads the last 5 sent issues from the database, passes their top items to a new
`synth.Roundup()` method with a longer-form prompt, and stores and publishes the result.
