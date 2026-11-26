#!/usr/bin/env python3
"""Deterministic quality checks for GoDaily's social copy.

Input: the JSON returned by the GoDaily MCP tool `social_post_metrics`
({"posts": [...], "total": N}). Large tool results are saved to a file by the
harness. Pass that file path directly.

    python3 -I .claude/skills/autopilot/evals/social_copy.py <posts.json> [--since YYYY-MM-DD] [--json]

Scores only what code can judge reliably. Grounding (claims that aren't in the
source item) needs the item snippets and an LLM judge: see rubric.md.

Stdlib only, so it runs in any cloud session without installing anything.
"""

from __future__ import annotations

import argparse
import difflib
import json
import re
import sys
from collections import defaultdict
from datetime import datetime, timedelta

# Phrases that #321 flagged as stock filler, plus common AI-default openers.
STOCK_PATTERNS = [
    r"\bworth (?:a |the )?(?:read(?:ing)?|track(?:ing)?|check(?:ing)? out|bookmark(?:ing)?|follow(?:ing)?|having|watch(?:ing)?|a look|your time|knowing)\b",
    r"\bwalks (?:you )?through\b",
    r"\bgame[- ]?changer\b",
    r"\bdeep[- ]dive\b",
    r"\bmust[- ]read\b",
    r"\bdive (?:in|into)\b",
    r"\bunlock(?:s|ing)? the\b",
    r"\bin today's (?:fast-paced|world)\b",
    r"\bseamless(?:ly)?\b",
    r"\bsupercharge\b",
]
STOCK_RE = [re.compile(p, re.IGNORECASE) for p in STOCK_PATTERNS]

URL_RE = re.compile(r"https?://\S+")
# Tracking/non-canonical URL markers (#319). godaily.dev's own utm_* links are fine.
TRACKING_RE = re.compile(r"(?:[?&](?:source=rss|utm_[a-z]+=|fbclid=|gclid=|ref=)|mailjet\.com|/d/msgid/)", re.IGNORECASE)

# Platform limits as of #330 (Mastodon counts every URL as 23 chars; Bluesky
# shows shortened URLs). Approximate, so they flag candidates rather than prove
# a problem.
LIMITS = {"bluesky": 300, "mastodon": 500, "linkedin": 3000}

STOPWORDS = set(
    "a an the and or of to in on for with from by is are was were be this that it its as at into your you "
    "go golang new how why what using use via about than more just not can".split()
)


def parse_time(s: str | None) -> datetime | None:
    if not s:
        return None
    s = s.replace("Z", "+00:00")
    # Trim nanoseconds that fromisoformat can't parse.
    s = re.sub(r"(\.\d{6})\d+", r"\1", s)
    try:
        return datetime.fromisoformat(s)
    except ValueError:
        return None


def platform_length(text: str, platform: str) -> int:
    if platform == "mastodon":
        return len(URL_RE.sub("x" * 23, text))
    if platform == "bluesky":
        def short(m: re.Match) -> str:
            u = re.sub(r"^https?://(www\.)?", "", m.group(0))
            return u[:30]
        return len(URL_RE.sub(short, text))
    return len(text)


def strip_noise(text: str) -> str:
    text = URL_RE.sub("", text)
    text = re.sub(r"#\w+", "", text)
    return re.sub(r"\s+", " ", text).strip().lower()


def keywords(text: str) -> set[str]:
    words = re.findall(r"[a-z0-9][a-z0-9.+/-]{2,}", strip_noise(text))
    return {w for w in words if w not in STOPWORDS}


def first_url(text: str) -> str | None:
    for u in URL_RE.findall(text):
        if "godaily.dev" in u:
            continue
        base, _, query = u.partition("?")
        keep = [kv for kv in query.split("&") if kv and not re.match(r"(utm_[a-z]+|source|ref|fbclid|gclid)=", kv)]
        return (base.rstrip("/") + ("?" + "&".join(keep) if keep else "")).lower()
    return None


def evaluate(posts: list[dict]) -> dict:
    n = len(posts)
    stock_hits: list[dict] = []
    tracking: list[dict] = []
    truncated: list[dict] = []
    over_limit: list[dict] = []
    footer = {"with": 0, "without": 0}
    by_group: dict[tuple, list[dict]] = defaultdict(list)

    for p in posts:
        text = p.get("text") or ""
        pid, platform, kind = p.get("id"), p.get("platform"), p.get("kind")
        for rx in STOCK_RE:
            m = rx.search(text)
            if m:
                stock_hits.append({"id": pid, "platform": platform, "kind": kind, "phrase": m.group(0)})
                break
        bad = [u for u in URL_RE.findall(text) if "godaily.dev" not in u and TRACKING_RE.search(u)]
        if bad:
            tracking.append({"id": pid, "platform": platform, "url": bad[0]})
        body = re.sub(r"(\s#\w+)+\s*$", "", URL_RE.sub("", text)).rstrip()
        if body.endswith("…") or body.endswith("..."):
            truncated.append({"id": pid, "platform": platform, "kind": kind})
        limit = LIMITS.get(platform)
        if limit and platform_length(text, platform) > limit:
            over_limit.append({"id": pid, "platform": platform, "length": platform_length(text, platform), "limit": limit})
        if kind in ("featured", "recap"):
            footer["with" if "subscribe" in text.lower() else "without"] += 1
        day = (p.get("posted_at") or "")[:10]
        by_group[(day, kind, p.get("subject") or p.get("issue_id"))].append(p)

    # Same text on every platform (#321): compare each pair within a group.
    cross_dupes = []
    for (day, kind, _), group in by_group.items():
        if len(group) < 2:
            continue
        for i in range(len(group)):
            for j in range(i + 1, len(group)):
                a, b = strip_noise(group[i].get("text", "")), strip_noise(group[j].get("text", ""))
                if not a or not b:
                    continue
                shorter, longer = sorted((a, b), key=len)
                ratio = difflib.SequenceMatcher(None, a, b).ratio()
                prefix = longer.startswith(shorter[: max(40, int(len(shorter) * 0.8))])
                if ratio >= 0.85 or prefix:
                    cross_dupes.append({
                        "day": day, "kind": kind,
                        "ids": [group[i]["id"], group[j]["id"]],
                        "platforms": [group[i]["platform"], group[j]["platform"]],
                        "similarity": round(ratio, 2), "prefix": prefix,
                    })

    # Same link or near-identical wording repeated within 14 days (#321). Semantic
    # repeats with different links (e.g. two JSON v2 videos) are for the LLM judge.: featured posts, one per day (LinkedIn
    # preferred as the longest text), compared on URL and keyword overlap.
    featured_by_day: dict[str, dict] = {}
    for p in posts:
        if p.get("kind") != "featured":
            continue
        day = (p.get("posted_at") or "")[:10]
        cur = featured_by_day.get(day)
        if cur is None or p.get("platform") == "linkedin":
            featured_by_day[day] = p
    days = sorted(featured_by_day)
    repeats = []
    for i, d1 in enumerate(days):
        t1 = parse_time(d1 + "T00:00:00+00:00")
        for d2 in days[i + 1:]:
            t2 = parse_time(d2 + "T00:00:00+00:00")
            if t1 and t2 and t2 - t1 > timedelta(days=14):
                break
            p1, p2 = featured_by_day[d1], featured_by_day[d2]
            k1, k2 = keywords(p1.get("text", "")), keywords(p2.get("text", ""))
            jacc = len(k1 & k2) / len(k1 | k2) if k1 and k2 else 0.0
            same_url = first_url(p1.get("text", "")) == first_url(p2.get("text", "")) and first_url(p1.get("text", ""))
            if same_url or jacc >= 0.30:
                repeats.append({"days": [d1, d2], "ids": [p1["id"], p2["id"]], "keyword_overlap": round(jacc, 2),
                                "same_url": bool(same_url), "shared": sorted(k1 & k2)[:8]})

    rate = lambda k: round(k / n, 3) if n else 0.0  # noqa: E731
    groups_multi = sum(1 for g in by_group.values() if len(g) >= 2)
    return {
        "posts": n,
        "scores": {
            "stock_phrase_rate": rate(len(stock_hits)),
            "tracking_url_rate": rate(len(tracking)),
            "truncated_rate": rate(len(truncated)),
            "over_limit_rate": rate(len(over_limit)),
            "cross_platform_dupe_rate": round(len(cross_dupes) / groups_multi, 3) if groups_multi else 0.0,
            "topic_repeats_14d": len(repeats),
            "subscribe_footer_rate": round(footer["with"] / (footer["with"] + footer["without"]), 3)
            if (footer["with"] + footer["without"]) else None,
        },
        "findings": {
            "stock_phrases": stock_hits,
            "tracking_urls": tracking,
            "truncated": truncated,
            "over_limit": over_limit,
            "cross_platform_dupes": cross_dupes,
            "topic_repeats": repeats,
        },
    }


def to_markdown(result: dict) -> str:
    s = result["scores"]
    lines = [
        f"**Social copy eval**: {result['posts']} posts (lower is better unless noted)",
        "",
        "| Check | Score |",
        "|---|---|",
        f"| Stock phrases (\"Worth …\" etc.) | {s['stock_phrase_rate']:.0%} of posts |",
        f"| Tracking / non-canonical URLs | {s['tracking_url_rate']:.0%} of posts |",
        f"| Truncated with … | {s['truncated_rate']:.0%} of posts |",
        f"| Over platform limit | {s['over_limit_rate']:.0%} of posts |",
        f"| Same text across platforms | {s['cross_platform_dupe_rate']:.0%} of multi-platform pairs |",
        f"| Featured link or wording repeated within 14 days | {s['topic_repeats_14d']} pairs |",
    ]
    if s["subscribe_footer_rate"] is not None:
        lines.append(f"| Featured/recap with Subscribe line (higher is better) | {s['subscribe_footer_rate']:.0%} |")
    return "\n".join(lines)


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("path", help="JSON file from social_post_metrics")
    ap.add_argument("--since", help="only posts with posted_at on/after this date (YYYY-MM-DD)")
    ap.add_argument("--json", action="store_true", help="print full JSON (scores + findings)")
    args = ap.parse_args()

    with open(args.path, encoding="utf-8") as f:
        data = json.load(f)
    posts = data.get("posts", data) if isinstance(data, dict) else data
    if args.since:
        posts = [p for p in posts if (p.get("posted_at") or "") >= args.since]

    result = evaluate(posts)
    print(json.dumps(result, indent=2) if args.json else to_markdown(result))
    return 0


if __name__ == "__main__":
    sys.exit(main())
