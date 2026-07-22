VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X github.com/octaspace/octa/cli.version=$(VERSION)

.PHONY: build clean test race vet fmt lint tidy check

build:
	go build -trimpath -ldflags="$(LDFLAGS)" -o octa ./cmd/octa/

clean:
	rm -f octa

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -l .

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

# Same gates the CI workflow runs.
check: build vet test race
	test -z "$$(gofmt -l .)"
