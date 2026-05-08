// Copyright (c) 2026 godaily (Ainsley Clark)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

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

// Config holds all environment variables consumed by the service.
// Optional fields are left empty if unset; callers guard against the zero value.
type Config struct {
	AppEnv                         env.Environment `env:"APP_ENV"`
	ResendToken                    string          `env:"RESEND_TOKEN,required"`
	AnthropicAPIKey                string          `env:"ANTHROPIC_API_KEY,required"`
	YouTubeAPIKey                  string          `env:"YOUTUBE_API_KEY"`
	GitHubToken                    string          `env:"GITHUB_TOKEN"`
	EmailSendAddress               string          `env:"EMAIL_SEND_ADDRESS,required"`
	TursoURL                       string          `env:"TURSO_URL,required"`
	TursoAuthToken                 string          `env:"TURSO_AUTH_TOKEN,required"`
	APISecret                      string          `env:"API_SECRET"`
	VercelDeployHookURL            string          `env:"VERCEL_DEPLOY_HOOK_URL"`
	BetterStackSendHeartbeatURL    string          `env:"BETTERSTACK_SEND_HEARTBEAT_URL"`
	BetterStackCollectHeartbeatURL string          `env:"BETTERSTACK_COLLECT_HEARTBEAT_URL"`
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
