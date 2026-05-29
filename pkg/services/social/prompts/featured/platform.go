// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package featured

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/ai"
	socialprompts "github.com/ainsleyclark/godaily/pkg/services/social/prompts"
	"github.com/ainsleyclark/godaily/pkg/util/aiutil"
)

// platformConfig captures the rules a single platform imposes on a post.
type platformConfig struct {
	name      string   // human label used in the prompt (e.g. "Bluesky")
	charLimit int      // hard character limit enforced after generation
	hashtags  []string // appended verbatim by the platform style rules
	guidance  string   // platform-specific voice + structure guidance
}

// reframe asks the model to recast the featured item as one platform post.
// The returned string is the raw text to send to the platform's API.
func reframe(ctx context.Context, p ai.Prompter, cfg platformConfig, f Featured) (string, error) {
	if p == nil {
		return "", errors.New("prompts: ai.Prompter is required")
	}
	if f.URL == "" {
		return "", errors.New("prompts: Featured.URL is required")
	}

	system := buildPlatformSystem(cfg)

	payload, err := json.Marshal(f)
	if err != nil {
		return "", errors.Wrap(err, "marshalling featured")
	}
	user := fmt.Sprintf(
		"Featured item to reframe:\n%s\n\nReturn the JSON object only.",
		string(payload),
	)

	raw, err := p.Prompt(ctx, system, user)
	if err != nil {
		return "", errors.Wrap(err, "ai")
	}

	text, err := parsePlatformPost(raw)
	if err != nil {
		return "", err
	}

	text = aiutil.SanitisePost(text)

	if n := utf8.RuneCountInString(text); n > cfg.charLimit {
		slog.Warn("Social post exceeded char limit; truncating",
			"platform", cfg.name, "chars", n, "limit", cfg.charLimit)
		text = aiutil.TruncatePost(text, cfg.charLimit)
	}
	return text, nil
}

func buildPlatformSystem(cfg platformConfig) string {
	var b strings.Builder
	fmt.Fprintf(&b, "You write a single social media post for %s in the voice of Ainsley Clark, a Go engineer.\n\n", cfg.name)
	b.WriteString("You will receive a JSON object describing today's featured item from the Go community (a release, proposal, article or similar). Recast it as ONE post that maximises engagement on this specific platform.\n\n")

	b.WriteString("## Platform constraints\n")
	fmt.Fprintf(&b, "- Maximum %d characters total (hard limit). Stay safely under it.\n", cfg.charLimit)
	b.WriteString("- Always include the item's URL verbatim. Never shorten or wrap it.\n")
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

	b.WriteString("## Platform-specific guidance\n")
	b.WriteString(cfg.guidance)
	b.WriteString("\n\n")

	b.WriteString("## Output format\n")
	b.WriteString("Output strict JSON, schema:\n")
	b.WriteString("{\n  \"text\": string   // the full post body, ready to submit verbatim\n}\n\n")
	b.WriteString("Output the JSON object alone. No prose, no markdown fences.")
	return b.String()
}

func parsePlatformPost(raw []byte) (string, error) {
	body := aiutil.StripFences(string(raw))
	if body == "" {
		return "", errors.New("empty platform post response")
	}
	var out struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		return "", fmt.Errorf("parse platform post (raw=%q): %w", body, err)
	}
	text := strings.TrimSpace(out.Text)
	if text == "" {
		return "", errors.New("platform post text is empty")
	}
	return text, nil
}
