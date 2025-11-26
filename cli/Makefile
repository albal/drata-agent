# Makefile for Drata Agent CLI

# Version
VERSION := 3.9.0-cli
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

# Binary name
BINARY := drata-agent

# Build directory
BUILD_DIR := build

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod

.PHONY: all build clean test deps help
.PHONY: build-linux build-linux-arm64 build-darwin build-darwin-arm64 build-windows
.PHONY: build-all install

all: deps test build

## help: Show this help message
help:
	@echo "Drata Agent CLI - Build Commands"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  all               Build for current platform (deps + test + build)"
	@echo "  build             Build for current platform"
	@echo "  build-all         Build for all platforms"
	@echo "  build-linux       Build for Linux (amd64)"
	@echo "  build-linux-arm64 Build for Linux (arm64)"
	@echo "  build-darwin      Build for macOS (amd64)"
	@echo "  build-darwin-arm64 Build for macOS (arm64/Apple Silicon)"
	@echo "  build-windows     Build for Windows (amd64)"
	@echo "  clean             Remove build artifacts"
	@echo "  deps              Download dependencies"
	@echo "  install           Install to GOPATH/bin"
	@echo "  test              Run tests"
	@echo ""

## deps: Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

## build: Build for current platform
build: deps
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) .

## build-linux: Build for Linux amd64
build-linux: deps
	mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-amd64 .

## build-linux-arm64: Build for Linux arm64
build-linux-arm64: deps
	mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-arm64 .

## build-darwin: Build for macOS amd64
build-darwin: deps
	mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 .

## build-darwin-arm64: Build for macOS arm64 (Apple Silicon)
build-darwin-arm64: deps
	mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 .

## build-windows: Build for Windows amd64
build-windows: deps
	mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe .

## build-all: Build for all platforms
build-all: build-linux build-linux-arm64 build-darwin build-darwin-arm64 build-windows
	@echo "Build complete. Binaries in $(BUILD_DIR)/"
	@ls -la $(BUILD_DIR)/

## test: Run tests
test:
	$(GOTEST) -v ./...

## clean: Clean build artifacts
clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)

## install: Install to GOPATH/bin
install: deps
	$(GOBUILD) $(LDFLAGS) -o $(GOPATH)/bin/$(BINARY) .
