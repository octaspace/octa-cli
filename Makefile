VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X github.com/octaspace/octa/cmd.version=$(VERSION)

build:
	go build -trimpath -ldflags="$(LDFLAGS)" -o octa .
