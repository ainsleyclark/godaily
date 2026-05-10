# Share Buttons on Digest Pages

Add lightweight share affordances to each digest page so readers can post to their network with a single click.

## Overview

Readers who enjoy an issue have no easy way to share it. A row of small share buttons beneath the digest heading reduces that friction and turns satisfied readers into organic promoters.

## Platforms

- **Copy link** — copies the canonical digest URL to clipboard
- **LinkedIn** — pre-filled share URL with the digest title
- **Bluesky** — intent URL opening the Bluesky compose window
- **X / Twitter** — tweet intent with title and URL

## Implementation

All share targets use platform-provided intent URLs (no API keys required). The copy-link button uses a small inline `navigator.clipboard` snippet. This is entirely a front-end change to the digest templ view with no backend involvement.

## Placement

A compact icon row sits directly beneath the issue subject line, above the first digest section.
