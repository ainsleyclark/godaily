
format: # Run gofmt
	go fmt ./...
.PHONY: format

lint: # Run linter
	golangci-lint run ./... --fix --config=.golangci.yaml
.PHONY: lint

cover: test # Run all the tests and opens the coverage report
	go tool cover -html=coverage.out
.PHONY: cover

lic: # Add license to all files
	find . -name "*.go" -type f -print0 | xargs -0 addlicense -c "godaily (Ainsley Clark)" -l bsd
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
