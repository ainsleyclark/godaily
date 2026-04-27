build: # Build
	go build -o godaily
.PHONY: build

generate: # Runs go generate
	go generate ./...
.PHONY: generate

run-dry: # Run godaily and write the aggregated digest to examples/rendered/news.json
	go run ./cmd/godaily run --dry-run --output examples/news.json
.PHONY: run

format: # Run gofmt
	go fmt ./...
.PHONY: format

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
	$(MAKE) test
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
