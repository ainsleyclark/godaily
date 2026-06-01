// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package env

import (
	"context"
	"os"

	"github.com/ainsleydev/webkit/pkg/env"
	cenv "github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// AppURL is the canonical production URL for godaily.
const AppURL = "https://godaily.dev"

// DashboardURL is the public base URL of the GoDaily admin dashboard.
// Hard-coded rather than threaded through config: there is exactly one
// dashboard per deployment and the URL never differs by environment.
const DashboardURL = "https://analytics.godaily.dev"

// Config holds all environment variables consumed by the service.
// Optional fields are left empty if unset; callers guard against the zero value.
type Config struct {
	AppEnv                                env.Environment `env:"APP_ENV"`
	ResendToken                           string          `env:"RESEND_TOKEN,required,unset"`
	ResendWebhookSecret                   string          `env:"RESEND_WEBHOOK_SECRET,unset"`
	AnthropicAPIKey                       string          `env:"ANTHROPIC_API_KEY,required,unset"`
	GeminiAPIKey                          string          `env:"GEMINI_API_KEY,unset"`
	YouTubeAPIKey                         string          `env:"YOUTUBE_API_KEY,required,unset"`
	GitHubToken                           string          `env:"GITHUB_TOKEN,unset"`
	ScraperAPIKeys                        []string        `env:"SCRAPER_API_KEY,unset"`
	EmailSendAddress                      string          `env:"EMAIL_SEND_ADDRESS,required,unset"`
	TursoURL                              string          `env:"TURSO_URL,required,unset"`
	TursoAuthToken                        string          `env:"TURSO_AUTH_TOKEN,required,unset"`
	APISecret                             string          `env:"API_SECRET,required,unset"`
	SlackToken                            string          `env:"SLACK_TOKEN,unset"`
	SlackChannel                          string          `env:"SLACK_CHANNEL,unset"`
	VercelDeployHookURL                   string          `env:"VERCEL_DEPLOY_HOOK_URL,unset"`
	BetterStackCollectHeartbeatURL        string          `env:"BETTERSTACK_COLLECT_HEARTBEAT_URL,unset"`
	BetterStackBuildHeartbeatURL          string          `env:"BETTERSTACK_BUILD_HEARTBEAT_URL,unset"`
	BetterStackSendHeartbeatURL           string          `env:"BETTERSTACK_SEND_HEARTBEAT_URL,unset"`
	BetterStackSocialFeaturedHeartbeatURL string          `env:"BETTERSTACK_SOCIAL_FEATURED_HEARTBEAT_URL,unset"`
	BetterStackSocialRotationHeartbeatURL string          `env:"BETTERSTACK_SOCIAL_ROTATION_HEARTBEAT_URL,unset"`
	BlueskyHandle                         string          `env:"BLUESKY_HANDLE,unset"`
	BlueskyAppPassword                    string          `env:"BLUESKY_APP_PASSWORD,unset"`
	LinkedInOAuthToken                    string          `env:"LINKEDIN_OAUTH_TOKEN,unset"`
	LinkedInOrgURN                        string          `env:"LINKEDIN_ORG_URN,unset"`
	MastodonServer                        string          `env:"MASTODON_SERVER,unset"`
	MastodonAppToken                      string          `env:"MASTODON_APP_TOKEN,unset"`
}

// New parses Config from the environment, overlaying values from a .env file
// in the working directory when present.
func New(_ context.Context) (Config, error) {
	// Vercel injects VERCEL=1 in all environments (dev, preview, production).
	// When present, env vars are provided by the platform, so we shouldn't
	// load the env file.
	isVercel := os.Getenv("VERCEL") != ""

	var cfg Config
	if !isVercel && env.IsDevelopment() {
		if err := godotenv.Load(".env"); err != nil {
			return cfg, err
		}
	}

	if err := cenv.Parse(&cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

// IsDevelopment returns whether we are running the app in development.
func (c Config) IsDevelopment() bool {
	return env.IsDevelopment()
}

// IsProduction returns whether we are running the app in production.
func (c Config) IsProduction() bool {
	return env.IsProduction()
}
