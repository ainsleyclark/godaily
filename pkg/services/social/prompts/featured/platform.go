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

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/compose"
	socialprompts "github.com/ainsleyclark/godaily/pkg/services/social/prompts"
	"github.com/ainsleyclark/godaily/pkg/util/aiutil"
)

// platformConfig captures the rules a single platform imposes on a post.
type platformConfig struct {
	platform  social.Platform // target platform; empty skips the Subscribe footer
	name      string          // human label used in the prompt (e.g. "Bluesky")
	charLimit int             // hard character limit enforced after generation
	hashtags  []string        // appended verbatim by the platform style rules
	guidance  string          // platform-specific voice + structure guidance
}

// reframe asks the model to recast the featured item as one platform post.
// The returned string is the raw text to send to the platform's API.
//
// Every featured post ends with a UTM-tagged Subscribe line so social
// sign-ups can be attributed (utm_campaign=featured). Its length is
// reserved up front: the model is told the smaller budget and any
// over-shoot is trimmed from the body, never the footer.
func reframe(ctx context.Context, p ai.Prompter, cfg platformConfig, f Featured) (string, error) {
	if p == nil {
		return "", errors.New("prompts: ai.Prompter is required")
	}
	if f.URL == "" {
		return "", errors.New("prompts: Featured.URL is required")
	}

	footer := ""
	if cfg.platform != "" {
		footer = compose.SubscribeLine(cfg.platform, string(social.PostKindFeatured))
	}
	budget := cfg.charLimit
	if footer != "" {
		budget -= compose.Length(cfg.platform, footer) + 2
	}

	system := buildPlatformSystem(cfg, budget)

	payload, err := json.Marshal(f)
	if err != nil {
		return "", errors.Wrap(err, "marshalling featured")
	}
	user := fmt.Sprintf(
		"Featured item to reframe:\n%s\n\nReturn the JSON object only.",
		string(payload),
	)

	raw, err := p.Prompt(ctx, ai.ModelSonnet, system, user)
	if err != nil {
		return "", errors.Wrap(err, "ai")
	}

	text, err := parsePlatformPost(raw)
	if err != nil {
		return "", err
	}

	// Em dashes are stripped centrally by ai.Client.Prompt before the
	// response reaches us, so no SanitisePost call is needed here.
	if n := compose.Length(cfg.platform, text); n > budget {
		slog.Warn("Social post exceeded char limit; truncating",
			"platform", cfg.name, "chars", n, "limit", budget)
		text = compose.FitTo(cfg.platform, text, budget)
	}
	if footer != "" {
		text += "\n\n" + footer
	}
	return text, nil
}

func buildPlatformSystem(cfg platformConfig, budget int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "You write a single social media post for %s in the voice of Ainsley Clark, a Go engineer.\n\n", cfg.name)
	b.WriteString("You will receive a JSON object describing today's featured item from the Go community (a release, proposal, article or similar). Recast it as ONE post that maximises engagement on this specific platform.\n\n")

	b.WriteString("## Platform constraints\n")
	fmt.Fprintf(&b, "- Maximum %d characters total (hard limit). Stay safely under it.\n", budget)
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
	body := aiutil.ExtractJSON(string(raw))
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
