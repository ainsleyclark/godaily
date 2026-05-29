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
SLACK_CHANNEL="#godaily-ingest"
SLACK_TOKEN="xoxb-REPLACE-ME"

log() { echo "==> $*"; }

slack_notify() {
	local text="$1"
	curl -fsS -X POST https://slack.com/api/chat.postMessage \
		-H "Authorization: Bearer ${SLACK_TOKEN}" \
		-H "Content-Type: application/json; charset=utf-8" \
		--data "$(printf '{"channel":"%s","text":%s}' "$SLACK_CHANNEL" "$(printf '%s' "$text" | jq -Rs .)")" \
		>/dev/null || log "Slack notification failed"
}

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
	slack_notify ":x: GoDaily Reddit ingest failed — Reddit returned an unexpected response (likely blocked)."
	exit 1
fi

# 2. Post the raw JSON to the ingest endpoint.
log "Posting to ${API_URL}/api/ingest/reddit/"
response="$(mktemp)"
trap 'rm -f "$tmp" "$response"' EXIT
http_code="$(curl -sS -o "$response" -w '%{http_code}' -X POST "${API_URL}/api/ingest/reddit/" \
	-H "Authorization: Bearer ${API_SECRET}" \
	-H "Content-Type: application/json" \
	--data-binary @"$tmp")"
body="$(cat "$response")"
echo "$body"
echo

if [[ "$http_code" != 2* ]]; then
	slack_notify ":x: GoDaily Reddit ingest failed (HTTP ${http_code}): ${body}"
	exit 1
fi

slack_notify ":white_check_mark: GoDaily Reddit ingest succeeded: ${body}"
log "Done"
