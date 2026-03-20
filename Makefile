BINARY  := krypt
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build test clean install cross

## build: compile for the current platform
build:
	go build $(LDFLAGS) -o $(BINARY) .

## test: run all unit tests
test:
	go test ./...

## install: install binary into $GOPATH/bin (or ~/go/bin)
install:
	go install $(LDFLAGS) .

## cross: compile for Linux, macOS (arm64 + amd64) and Windows
cross:
	GOOS=linux   GOARCH=amd64  go build $(LDFLAGS) -o dist/$(BINARY)-linux-amd64   .
	GOOS=linux   GOARCH=arm64  go build $(LDFLAGS) -o dist/$(BINARY)-linux-arm64   .
	GOOS=darwin  GOARCH=amd64  go build $(LDFLAGS) -o dist/$(BINARY)-darwin-amd64  .
	GOOS=darwin  GOARCH=arm64  go build $(LDFLAGS) -o dist/$(BINARY)-darwin-arm64  .
	GOOS=windows GOARCH=amd64  go build $(LDFLAGS) -o dist/$(BINARY)-windows-amd64.exe .

## clean: remove built artifacts
clean:
	rm -f $(BINARY)
	rm -rf dist/
