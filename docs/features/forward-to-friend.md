# Forward-to-Friend & Referral CTA

Grow the subscriber list passively by turning every forwarded email into a potential new signup.

## Overview

A significant portion of newsletter growth comes from forwarded emails. GoDaily currently has no CTA for non-subscribers who receive a forwarded copy. Adding a single line to the email footer converts this lost traffic.

## Basic CTA (minimal effort)

Add to the email footer:

> Was this forwarded to you? [Subscribe to GoDaily →](https://godaily.dev)

This requires a one-line change to the email template and no backend work.

## Referral Tracking (optional extension)

Each subscriber is given a unique share link (`https://godaily.dev?ref=<token>`) that they can include when forwarding or posting about the digest. When a new subscriber signs up via a referral link, the referring subscriber is credited.

Referral counts could surface as a simple leaderboard or be used to reward engaged subscribers (e.g. early access to new features).

### Data needed
- `referral_token` column on the `subscribers` table
- `referred_by` foreign key on new signups
- A query to count referrals per subscriber
