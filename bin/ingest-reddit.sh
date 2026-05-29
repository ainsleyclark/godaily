#!/usr/bin/env bash
set -euo pipefail

# Fetches the r/golang "new" listing from the machine this script runs on (whose
# residential IP Reddit does not block) and POSTs it to the GoDaily ingest
# endpoint, which transforms and persists the items for the current collection
# window. Intended to run on a schedule (e.g. cron) as a fallback while the
# server-side Reddit fetch is blocked. The endpoint de-duplicates on (url, tag),
# so running it repeatedly is safe.
#
# Required env:
#   GODAILY_API_SECRET   Bearer token (matches the server's API_SECRET).
# Optional env:
#   GODAILY_API_URL      Base URL of the API (default: https://godaily.dev).
#   REDDIT_URL           Listing to fetch (default: r/golang/new, limit 25).
#   REDDIT_USER_AGENT    User-Agent sent to Reddit (Reddit blocks generic ones).

API_URL="${GODAILY_API_URL:-https://godaily.dev}"
API_SECRET="${GODAILY_API_SECRET:?GODAILY_API_SECRET must be set}"
REDDIT_URL="${REDDIT_URL:-https://www.reddit.com/r/golang/new.json?limit=25}"
USER_AGENT="${REDDIT_USER_AGENT:-godaily-ingest/1.0 (https://godaily.dev)}"

log() { echo "==> $*"; }

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT

# 1. Fetch the listing from Reddit directly (no proxy — runs on your IP).
log "Fetching ${REDDIT_URL}"
curl -fsSL -A "$USER_AGENT" "$REDDIT_URL" -o "$tmp"

# Sanity-check the payload before posting: must look like a Reddit listing.
if ! grep -q '"children"' "$tmp"; then
	log "Unexpected response (no \"children\" array) — Reddit may have blocked this request:"
	head -c 500 "$tmp" >&2
	echo >&2
	exit 1
fi

# 2. Post the raw JSON to the ingest endpoint.
log "Posting to ${API_URL}/api/ingest/reddit"
curl -fsS -X POST "${API_URL}/api/ingest/reddit" \
	-H "Authorization: Bearer ${API_SECRET}" \
	-H "Content-Type: application/json" \
	--data-binary @"$tmp"
echo

log "Done"
