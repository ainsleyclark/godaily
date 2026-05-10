# Subscriber Count / Social Proof

Display a live subscriber count on the homepage to provide social proof and improve signup conversion.

## Overview

"Join 1,200 Go developers" is one of the most effective one-line conversion improvements a newsletter can make. The data already exists in the `subscribers` table — it just needs to be surfaced on the homepage.

## Approach

Add a `CountActiveSubscribers` query to the store layer and pass the result to the homepage handler. The homepage templ template is updated to render the count beneath the subscribe form.

## Display Options

- Exact count: "Join 1,247 subscribers"
- Rounded count: "Join 1,200+ developers" (avoids looking small in early days)
- Conditional: only show once a threshold (e.g. 100) is reached

## Caching

A live DB query on every homepage load is fine at low traffic. If load increases, the count can be cached in memory and refreshed every few minutes.
