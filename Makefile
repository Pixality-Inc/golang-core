.PHONY: all
all: dep gen lint test

.PHONY: gen
gen:
	go generate ./...

.PHONY: dep
dep:
	go mod tidy
	go mod download

.PHONY: test
test:
	go test ./...

.PHONY: lint
lint:
	env GOCACHE=/tmp/go-build GOLANGCI_LINT_CACHE=/tmp/golangci-lint-cache golangci-lint run --tests=true ./...
