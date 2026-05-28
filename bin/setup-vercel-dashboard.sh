#!/usr/bin/env bash
# One-time setup for the godaily-dashboard Vercel project (analytics.godaily.dev).
#
# Idempotent: re-running will skip steps the Vercel CLI rejects (e.g. env var
# already set, domain already attached).
#
# Prerequisite: `npm i -g vercel` and `vercel login`.
#
# The CLI cannot set Root Directory, Framework Preset, or Ignored Build Step —
# those are project-creation settings only modifiable via the Vercel dashboard
# or REST API. The checklist at the end of this script prints what to click.

set -euo pipefail

log() { echo "==> $*"; }

if ! command -v vercel &> /dev/null; then
	echo "Vercel CLI not found. Install with: npm i -g vercel" >&2
	exit 1
fi

cd "$(git rev-parse --show-toplevel)/dashboard"

log "Linking dashboard/ to a Vercel project (create new if prompted)"
vercel link

log "Setting PUBLIC_API_BASE_URL for production"
echo "https://godaily.dev" | vercel env add PUBLIC_API_BASE_URL production || \
	log "  (env var already set — skipping)"

log "Attaching analytics.godaily.dev"
vercel domains add analytics.godaily.dev || \
	log "  (domain already attached — skipping)"

cat <<'EOF'

==> CLI steps complete. Finish in the Vercel dashboard:

  Project Settings → General
    Root Directory       : dashboard
  (Framework, Build / Install / Output are set by dashboard/vercel.json —
   leave the UI Override toggles OFF for those.)

  Project Settings → Git → Ignored Build Step
    Command: git diff --quiet HEAD^ HEAD .
    (exit 0 = skip; exit 1 = build — runs from the Root Directory)

  Domains
    Promote analytics.godaily.dev to Production (assign to main branch).

  For the EXISTING godaily project (godaily.dev), also set its Ignored Build Step to:
    git diff --quiet HEAD^ HEAD ':(exclude)dashboard'
EOF
