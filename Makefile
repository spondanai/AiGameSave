BINARY_NAME := ags
INSTALL_PATH := $(shell go env GOPATH)/bin/$(BINARY_NAME)
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build install dev test fmt lint clean help

build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/ags

install:
	go install $(LDFLAGS) ./cmd/ags
	@echo "Installed: $(INSTALL_PATH)"

dev:
	go run ./cmd/ags $(ARGS)

test:
	go test ./...

fmt:
	go fmt ./...

lint:
	go vet ./...

clean:
	rm -f bin/$(BINARY_NAME)

help:
	@echo "Usage:"
	@echo "  make build    Build binary to bin/ags"
	@echo "  make install  Install to GOPATH/bin (run this after pulling changes)"
	@echo "  make dev      Run without building  (ARGS='save' or ARGS='load --stdout')"
	@echo "  make test     Run tests"
	@echo "  make fmt      Format code"
	@echo "  make lint     Vet code"
	@echo "  make clean    Remove bin/ags"
