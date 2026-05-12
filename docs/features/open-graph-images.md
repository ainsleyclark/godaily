# Open Graph Images

Generate a branded social preview image for each digest so that sharing a digest URL on LinkedIn,
Slack, or Bluesky produces a rich visual card rather than a blank link.

## Overview

Each digest page currently has no `og:image`. Adding a dynamic image endpoint means every organic
share becomes a branded touchpoint with zero extra effort from the poster.

I have created the HTML designs to be used in ~/Downloads/GoDaily/OG Images.html

The file contains two designs.

1) Generic homepage open graph image or one to be used when one isn't defined (such as privacy
   page).
2) A dynamic open graph image for the issues that contains the title etc.

## Approach

We could tackle this in two seperate ways. I'm leaning towards the pure Go version so it's in
keeping with the rest of the repo.

### Vercel

Use vercel opengraph image generation here: https://vercel.com/docs/og-image-generation Using JS

### Go

Programmatically generate them, we can take inspiration
from https://pace.dev/blog/2020/03/02/dynamically-generate-social-images-in-golang-by-mat-ryer.html
by perhaps using https://pkg.go.dev/github.com/fogleman/gg

These would be generated at generate time using web/generate. Perhaps we could create a new package
called web/og.

But feel free to suggest and ask things. Make sure you read AGENTS.md

## Caching

Images are expensive to generate repeatedly. The handler should set a long `Cache-Control` header
and optionally store generated PNGs in the database or object storage so repeated requests are
served instantly.
