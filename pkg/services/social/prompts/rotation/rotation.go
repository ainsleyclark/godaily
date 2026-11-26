// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rotation

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/compose"
	socialprompts "github.com/ainsleyclark/godaily/pkg/services/social/prompts"
	"github.com/ainsleyclark/godaily/pkg/services/social/prompts/featured"
	"github.com/ainsleyclark/godaily/pkg/util/aiutil"
)

// platformProfile carries the per-platform tone and hashtag list. Values
// mirror the featured-path constants in pkg/services/social/prompts so the
// rotation feed reads the same as the daily slot. Char limits live in
// pkg/services/social/compose.
type platformProfile struct {
	name     string
	hashtags []string
	guidance string
}

// platformProfiles maps each platform to its rotation rules. Hashtag
// lists are copied from the featured-path prompts so a reader can't tell
// which slot a post came from.
var platformProfiles = map[social.Platform]platformProfile{
	social.Bluesky: {
		name:     "Bluesky",
		hashtags: featured.BlueskyHashtags,
		guidance: `- Bluesky users are heavily developer-focused. Speak like you're posting in a Go channel.
- Drop bare URLs on their own line — Bluesky linkifies them automatically. No markdown.
- 200-280 chars is the sweet spot.`,
	},
	social.LinkedIn: {
		name:     "LinkedIn",
		hashtags: featured.LinkedInHashtags,
		guidance: `- The audience is engineering leaders and senior developers. Plain prose paragraphs, no bullet lists, no markdown.
- 300-600 chars is the sweet spot. The hard limit is much higher; do NOT pad.`,
	},
	social.Mastodon: {
		name:     "Mastodon",
		hashtags: featured.MastodonHashtags,
		guidance: `- The fediverse uses hashtags actively for discovery — keep them.
- Drop the URL on its own line; Mastodon renders it clickably.
- 280-400 chars is the sweet spot.`,
	},
}

// run executes one generate-and-parse cycle: it formats the kind-specific
// system prompt with the platform's rules, calls the AI, parses the
// {"text": "..."} JSON, and appends footer (e.g. the Subscribe line).
//
// The footer's length is reserved up front: the model is told the
// smaller budget, and any over-shoot is trimmed from the body (whole
// lines first) so the footer always survives and platform APIs (e.g.
// Bluesky's 300-grapheme cap) never reject the post.
func run(
	ctx context.Context,
	p ai.Prompter,
	platform social.Platform,
	kindSystem string,
	userPayload any,
	footer string,
) (string, error) {
	if p == nil {
		return "", errors.New("rotation: ai.Prompter is required")
	}
	cfg, ok := platformProfiles[platform]
	if !ok {
		return "", errors.Errorf("rotation: unknown platform %q", platform)
	}

	system := assembleSystem(cfg, bodyBudget(platform, footer), kindSystem)

	user, err := buildUser(userPayload)
	if err != nil {
		return "", err
	}

	raw, err := p.Prompt(ctx, ai.ModelSonnet, system, user)
	if err != nil {
		return "", errors.Wrap(err, "ai")
	}

	text, err := parseTextResponse(raw)
	if err != nil {
		return "", err
	}

	// Em dashes are stripped centrally by ai.Client.Prompt before the
	// response reaches us, so no SanitisePost call is needed here.
	if n, budget := compose.Length(platform, text), bodyBudget(platform, footer); n > budget {
		slog.Warn("Rotation post exceeded char limit; trimming",
			"platform", cfg.name, "chars", n, "budget", budget)
	}
	return compose.WithFooter(platform, text, footer), nil
}

// bodyBudget is the number of characters left for the post body once the
// footer (and the blank line separating it) is reserved.
func bodyBudget(platform social.Platform, footer string) int {
	limit := compose.Limit(platform)
	if footer == "" {
		return limit
	}
	return limit - compose.Length(platform, footer) - 2
}

// assembleSystem builds the platform constraints + voice section that
// every rotation kind wears around its kind-specific guidance.
func assembleSystem(cfg platformProfile, budget int, kindGuidance string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "You write one social media post for %s in the voice of Ainsley Clark, a Go engineer.\n\n", cfg.name)

	b.WriteString("## Platform constraints\n")
	fmt.Fprintf(&b, "- Maximum %d characters total (hard limit). Stay safely under it.\n", budget)
	if len(cfg.hashtags) > 0 {
		b.WriteString("- End with these hashtags exactly, in order, on the final line:\n  ")
		b.WriteString(strings.Join(cfg.hashtags, " "))
		b.WriteString("\n")
	} else {
		b.WriteString("- No hashtags. The platform does not use them effectively.\n")
	}
	b.WriteString("\n")

	b.WriteString(socialprompts.BrandRules)
	b.WriteString("\n")

	b.WriteString("## Platform guidance\n")
	b.WriteString(cfg.guidance)
	b.WriteString("\n\n")

	b.WriteString("## This post\n")
	b.WriteString(kindGuidance)
	b.WriteString("\n\n")

	b.WriteString("## Output format\n")
	b.WriteString("Output strict JSON, schema:\n{\n  \"text\": string   // the full post body, ready to submit verbatim\n}\n\n")
	b.WriteString("Output the JSON object alone. No prose, no markdown fences.")
	return b.String()
}

func buildUser(payload any) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", errors.Wrap(err, "marshalling user payload")
	}
	return fmt.Sprintf("Inputs:\n%s\n\nReturn the JSON object only.", string(body)), nil
}

func parseTextResponse(raw []byte) (string, error) {
	body := aiutil.ExtractJSON(string(raw))
	if body == "" {
		return "", errors.New("empty rotation response")
	}
	var out struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		return "", fmt.Errorf("parse rotation post (raw=%q): %w", body, err)
	}
	text := strings.TrimSpace(out.Text)
	if text == "" {
		return "", errors.New("rotation post text is empty")
	}
	return text, nil
}
