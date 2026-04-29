BINARY_NAME := ags
INSTALL_PATH := $(shell go env GOPATH)/bin/$(BINARY_NAME)
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"
ROOT_DIR := $(shell pwd)

.PHONY: build install dev dev-install save load test fmt lint clean help

build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/ags

install:
	go install $(LDFLAGS) ./cmd/ags
	@echo "Installed: $(INSTALL_PATH)"

dev:
	go run ./cmd/ags $(ARGS)

dev-install:
	@mkdir -p "$(dir $(INSTALL_PATH))"
	@printf '#!/bin/sh\ncd "%s" || exit 1\nexec go run ./cmd/ags "$$@"\n' "$(ROOT_DIR)" > "$(INSTALL_PATH)"
	@chmod +x "$(INSTALL_PATH)"
	@echo "Installed dev wrapper: $(INSTALL_PATH)"

save:
	go run ./cmd/ags save

load:
	go run ./cmd/ags load $(ARGS)

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
	@echo "  make save     Run ags save from latest source"
	@echo "  make load     Run ags load from latest source (ARGS='--stdout')"
	@echo "  make dev-install  Replace ags with a dev wrapper that always runs this repo"
	@echo "  make test     Run tests"
	@echo "  make fmt      Format code"
	@echo "  make lint     Vet code"
	@echo "  make clean    Remove bin/ags"
