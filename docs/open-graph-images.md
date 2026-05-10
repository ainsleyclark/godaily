# Open Graph Images

Generate a branded social preview image for each digest so that sharing a digest URL on LinkedIn, Slack, or Bluesky produces a rich visual card rather than a blank link.

## Overview

Each digest page currently has no `og:image`. Adding a dynamic image endpoint means every organic share becomes a branded touchpoint with zero extra effort from the poster.

## Approach

A new handler serves a PNG at `/api/og/:slug` that is generated on the fly using Go's standard image libraries. The digest page templates are updated to reference this URL in their `<meta property="og:image">` tag.

## Image Content

- GoDaily logo / wordmark
- Digest date
- Top headline (truncated to fit)
- Subtle background matching the site palette

## Caching

Images are expensive to generate repeatedly. The handler should set a long `Cache-Control` header and optionally store generated PNGs in the database or object storage so repeated requests are served instantly.
