.PHONY: all
all: dep lint test

.PHONY: dep
dep:
	go mod tidy
	go mod download

.PHONY: test
test:
	go test ./...

.PHONY: lint
lint:
	golangci-lint run --tests
