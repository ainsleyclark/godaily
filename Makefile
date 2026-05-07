build: # Build
	go build -o godaily
.PHONY: build

generate: # Runs go generate
	go generate ./...
	go run main.go run --dry-run --output examples/news.json
.PHONY: generate

run: # Sends the godaily email
	go run main.go run
.PHONY: run

serve: # Start live-reload dev environment (templ + air + esbuild). Visit http://localhost:3000
	@mkdir -p tmp
	@command -v pnpm >/dev/null 2>&1 || { echo "pnpm is required (npm i -g pnpm)"; exit 1; }
	@test -d web/node_modules || (cd web && pnpm install)
	web/node_modules/.bin/concurrently -k \
		-n templ,air,esbuild \
		-c blue,green,magenta \
		"go tool templ generate --watch --path=./web" \
		"go tool air -c .air.toml" \
		"pnpm --dir web dev"
.PHONY: serve

serve-prod: # Start the HTTP web server without live-reload
	go run main.go serve
.PHONY: serve-prod

build-static: # Build static site into out/ (mirrors Vercel's install + build pipeline)
	bash bin/install.sh && bash bin/build.sh
.PHONY: build-static

run-dry: # Run godaily and write the aggregated digest to examples/rendered/news.json
	go run main.go run --dry-run --output examples/news.json
.PHONY: run

format: # Run gofmt
	go fmt ./...
.PHONY: format

gen: # Runs all //go:generate
	go generate ./...
.PHONY: gen

sqlc: # Regenerate sqlc output from internal/store/*.sql and the migrations
	sqlc generate
.PHONY: sqlc

migrate-up: # Apply all pending database migrations against TURSO_URL
	go run main.go migrate up
.PHONY: migrate-up

migrate-down: # Roll back the most recent database migration against TURSO_URL
	go run main.go migrate down
.PHONY: migrate-down

excluded := grep -v gen | grep -v res

test: # Test uses race and coverage
	go clean -testcache && go test $$(go list ./... | $(excluded)) -coverprofile=coverage.out -covermode=atomic
.PHONY: test

test-race: # Test uses race and coverage
	go clean -testcache && go test -race $$(go list ./... | $(excluded)) -coverprofile=coverage.out -covermode=atomic
.PHONY: test-race

test-integration: # Run integration tests against real source endpoints
	go test -v -tags=integration -run TestSources_Integration ./internal/source/...
.PHONY: test-sources

lint: # Run linter
	golangci-lint run ./... --fix --config=.golangci.yaml
.PHONY: lint

sec: # Run gosec security scan (matches CI)
	@command -v gosec >/dev/null 2>&1 || go install github.com/securego/gosec/v2/cmd/gosec@latest
	gosec -exclude-generated ./...
.PHONY: sec

vuln: # Run govulncheck (matches CI)
	@command -v govulncheck >/dev/null 2>&1 || go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...
.PHONY: vuln

cover: test # Run all the tests and opens the coverage report
	go tool cover -html=coverage.out
.PHONY: cover

lic: # Add license to all files
	find . -name "*.go" -type f -print0 | xargs -0 perl -0777 -i -pe 's|^// Copyright[^\n]*\n(//[^\n]*\n)*\n?||'
	find . -name "*.go" -type f -print0 | xargs -0 addlicense -c "godaily (Ainsley Clark)" -l mit
.PHONY: lic

doc: # Run go doc
	godoc -http localhost:8080
.PHONY: doc

all: # Make format, lint and test
	$(MAKE) lic
	$(MAKE) format
	$(MAKE) lint
	$(MAKE) test-race
	$(MAKE) vuln
	$(MAKE) sec
.PHONY: all

todo: # Show to-do items per file
	$(Q) grep \
		--exclude=Makefile.util \
		--exclude-dir=vendor \
		--exclude-dir=.vercel \
		--exclude-dir=.gen \
		--exclude-dir=.idea \
		--exclude-dir=public \
		--exclude-dir=node_modules \
		--exclude-dir=archetypes \
		--exclude-dir=.git \
		--text \
		--color \
		-nRo \
		-E '\S*[^\.]TODO.*' \
		.
.PHONY: todo

help: # Display this help
	$(Q) awk 'BEGIN {FS = ":.*#"; printf "Usage: make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?#/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
.PHONY: help
