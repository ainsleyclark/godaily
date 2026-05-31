# Share & Channel Attribution

GoDaily already shares itself widely — the issue pages carry LinkedIn,
Bluesky, and X share buttons, and the social service auto-posts to
several platforms every day. What's missing is the ability to tell which
of those efforts actually brings people in. Every shared link today is
anonymous, so the funnel ends in a question mark.

## Why this exists

You cannot improve what you cannot see. GoDaily collects rich engagement
data — opens, clicks, the path each issue takes — but the moment a link
leaves for an external platform, the trail goes cold. When a new
subscriber appears, there is no way to know whether they came from a
Bluesky post, a reader who forwarded the email, a search result, or the
homepage. Effort gets spread evenly across every channel because none of
them can be told apart.

This is the cheapest high-value change available: it does not try to grow
the audience directly, it makes every *other* growth effort measurable.
At the current stage, knowing where subscribers come from is worth more
than any single tactic, because it tells you where to spend next.

## The end result

- **Every channel becomes distinguishable.** A subscribe that originated
  from the daily Bluesky post, a shared issue link, or the homepage can
  be told apart from the others.
- **The existing metrics finally answer "where from?"** The data GoDaily
  already records can be rolled up by channel, so the dashboard shows not
  just how many people subscribed, but through which door they walked.
- **Spend follows evidence.** Channels that reliably convert get more
  attention; channels that look busy but bring nobody in can be quietly
  retired. Decisions stop being guesses.
- **A foundation for everything later.** Referral programs, social
  experiments, and campaign ideas all depend on being able to measure a
  channel. This is the prerequisite that unlocks them.

## What success looks like

For any given week, GoDaily can state with confidence which two or three
channels produced the most new confirmed subscribers, and that statement
is backed by recorded data rather than intuition. The first time a growth
idea is tried, its impact is visible instead of inferred.

## What this is not

This is not surveillance of readers and it does not compromise the "no
tracking pixels" promise. The aim is to understand which *channels* work,
at an aggregate level — not to follow individuals around the web.

## How it works

Attribution rides entirely on **Plausible**, which GoDaily already loads
in production. Plausible reads standard `utm_*` query parameters off the
landing URL and rolls visits up by source — so the whole mechanism is two
small parts: tag the links we send out, and tell Plausible when a signup
happens. No database columns, no per-reader tracking.

**Tagged links.** Every GoDaily-owned link that leaves the product carries
UTM parameters naming the channel it left through. The tagging is
centralised in `pkg/utm` (`utm.Tag(url, source, medium, campaign)`):

| Surface | utm_source | utm_medium | utm_campaign |
| --- | --- | --- | --- |
| Email digest issue link | `email` | `email` | `daily-digest` |
| Share button — LinkedIn | `linkedin` | `share` | `issue-share` |
| Share button — Bluesky | `bluesky` | `share` | `issue-share` |
| Share button — X / Twitter | `twitter` | `share` | `issue-share` |
| Share button — copy link | `copy` | `share` | `issue-share` |
| Auto social CTA post | `social-<platform>` | `social` | `cta` |

**The conversion.** When the homepage subscribe form submits successfully,
the frontend fires a Plausible custom event, `Signup`, in the *same browser
session* as the landing — so Plausible attributes it to the original
`utm_source`. The double opt-in confirmation happens in a later session and
is deliberately not the tracked event; tying it back would need plumbing
that buys little over the submit signal.

**One manual step.** Create a custom-event goal named **`Signup`** in the
Plausible dashboard. Until that goal exists the event is sent but not
counted; once it does, the goal can be broken down by source and campaign,
and the question "where from?" finally has an answer.

**What stays untagged.** The featured, recap, spotlight and new-source
auto-posts link to *external* articles, not back to GoDaily, so they carry
no UTM tags — there is nothing on the GoDaily side for them to attribute,
and tagging someone else's URL would be rude noise.
