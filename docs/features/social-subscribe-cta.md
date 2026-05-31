# Social posts: subscribe CTA on every post

## Problem

Featured, recap, new_source, spotlight, and community posts contain no link back
to `godaily.dev`. Someone who discovers GoDaily through a social post has no
obvious next step.

Only the dedicated `cta` rotation slot (fires every ~2 weeks on Tuesdays)
includes a subscribe URL. All other post types leave the acquisition path closed.

## Approach

Post-process the AI-generated text in `post.go` (featured path) and
`rotation.go` (rotation path) to append a UTM-tagged subscribe line after
generation. This is deterministic, requires no type-signature changes, and
degrades gracefully when headroom is tight.

### Per-platform behaviour

| Platform | Action |
|---|---|
| **LinkedIn** (1300 chars) | Append subscribe line — always fits |
| **Mastodon** (500 chars) | Append subscribe line if result ≤ 500 chars; otherwise skip silently |
| **Bluesky** (300 chars) | Skip — 300-char limit leaves no headroom after a full post + URL + hashtags |

### Subscribe line format

```
\n\nSubscribe: https://godaily.dev/?utm_source=social-{platform}&utm_medium=social&utm_campaign={kind}
```

`campaign` = the `PostKind` string: `featured`, `recap`, `new_source`,
`spotlight`, `community`. This lets Plausible attribute new subscribers to the
exact post type that converted them.

`cta` posts are excluded — they are already a subscribe CTA.

## Implementation

### 1. New helper — `pkg/services/social/subscribe.go`

```go
// appendSubscribeLine appends a UTM-tagged GoDaily subscribe URL after the
// post text. Bluesky is skipped (300-char limit leaves no headroom).
// If appending would exceed the platform char limit, the original text is
// returned unchanged.
func appendSubscribeLine(text string, plat social.Platform, campaign string) string {
    charLimits := map[social.Platform]int{
        social.LinkedIn: 1300,
        social.Mastodon: 500,
    }
    limit, ok := charLimits[plat]
    if !ok {
        return text
    }
    subscribeURL := utm.Tag(env.AppURL+"/", "social-"+plat.String(), "social", campaign)
    full := text + "\n\nSubscribe: " + subscribeURL
    if utf8.RuneCountInString(full) > limit {
        return text
    }
    return full
}
```

### 2. `pkg/services/social/post.go` — `generate` closure

```go
text, err := reframe(ctx, s.prompter, feat)
if err != nil {
    return "", err
}
return appendSubscribeLine(text, p, string(social.PostKindFeatured)), nil
```

### 3. `pkg/services/social/rotation.go` — `generate` closure

```go
text, err := cand.Generate(ctx, s.prompter, p, cctx)
if err != nil {
    return "", err
}
if cctx.Kind != social.PostKindCTA {
    text = appendSubscribeLine(text, p, string(cctx.Kind))
}
return text, nil
```

## Testing

- `go test ./pkg/services/social/...`
- Verify Bluesky posts are returned unchanged.
- Verify LinkedIn/Mastodon posts have the subscribe line appended with correct UTM params.
- Verify a post that exactly fills the Mastodon char limit is returned unchanged.
