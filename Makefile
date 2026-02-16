.PHONY: help build run clean install test vet ci-test ci-build release-build package release
.DEFAULT_GOAL := help

# ===================================================================
# CONFIGURATION
# ===================================================================

# Project settings
BINARY_NAME := LumpfishMediaCatcher
PACKAGE_NAME := LumpfishMediaCatcher
CMD_PATH := ./cmd/lumpfishmediacatcher

# Version (can be overridden via: make VERSION=v1.0.0 release-build)
VERSION ?= dev

# Build directory
BUILD_DIR := build
DIST_DIR := dist

# Platform detection
ifeq ($(OS),Windows_NT)
	DETECTED_OS := windows
	BINARY_EXT := .exe
	ARCHIVE_EXT := zip
else
	UNAME_S := $(shell uname -s)
	ifeq ($(UNAME_S),Linux)
		DETECTED_OS := linux
	else ifeq ($(UNAME_S),Darwin)
		DETECTED_OS := darwin
	endif
	BINARY_EXT :=
	ARCHIVE_EXT := tar.gz
endif

# Go build flags
GO := go
CGO_ENABLED := 1
LDFLAGS := -s -w -X main.version=$(VERSION)
BUILD_FLAGS := -v -ldflags="$(LDFLAGS)"

# Full binary name with extension
BINARY := $(BINARY_NAME)$(BINARY_EXT)

# Architecture (can be overridden)
GOARCH ?= amd64

# ===================================================================
# LOCAL DEVELOPMENT TARGETS (Simple & Fast)
# ===================================================================

## help: Show this help message
help:
	@echo 'Usage:'
	@echo '  make <target>'
	@echo ''
	@echo 'Local Development:'
	@echo '  build         Build the application for local use'
	@echo '  run           Build and run the application'
	@echo '  clean         Remove build artifacts'
	@echo '  test          Run tests'
	@echo '  install       Install Go dependencies'
	@echo ''
	@echo 'CI/CD:'
	@echo '  vet           Run go vet'
	@echo '  ci-test       Run all CI tests (test + vet + build)'
	@echo '  ci-build      Build binary (for CI verification)'
	@echo ''
	@echo 'Release:'
	@echo '  release-build Build versioned binary with ldflags'
	@echo '  package       Create distribution archive'
	@echo '  release       Full release build + package'
	@echo ''
	@echo 'Variables:'
	@echo '  VERSION       Version string for release builds (default: dev)'
	@echo '  GOARCH        Target architecture (default: amd64)'

## build: Build the application for local development (simple, no versioning)
build:
	@echo "Building $(BINARY_NAME) for local development..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build -o $(BUILD_DIR)/$(BINARY) $(CMD_PATH)
	@echo "Built: $(BUILD_DIR)/$(BINARY)"

## run: Build and run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY)

## clean: Remove all build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR)
	@echo "Clean complete"

## install: Download Go module dependencies
install:
	@echo "Installing dependencies..."
	$(GO) mod download
	@echo "Dependencies installed"

## test: Run all tests
test:
	@echo "Running tests..."
	$(GO) test -v ./...

# ===================================================================
# CI/CD TARGETS
# ===================================================================

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GO) vet ./...

## ci-test: Run all CI tests (test + vet + build verification)
ci-test: test vet ci-build
	@echo "All CI tests passed"

## ci-build: Build binary for CI verification (no packaging)
ci-build:
	@echo "Building for CI verification..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY) $(CMD_PATH)
	@echo "CI build complete: $(BUILD_DIR)/$(BINARY)"

# ===================================================================
# RELEASE TARGETS
# ===================================================================

## release-build: Build versioned binary with ldflags
release-build:
	@echo "Building $(BINARY_NAME) $(VERSION) for $(DETECTED_OS)-$(GOARCH)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(DETECTED_OS) GOARCH=$(GOARCH) \
		$(GO) build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY) $(CMD_PATH)
	@echo "Release build complete: $(BUILD_DIR)/$(BINARY)"
	@echo "Version: $(VERSION)"

## package: Create distribution archive (requires release-build to run first)
package:
	@echo "Creating distribution package..."
	@mkdir -p $(DIST_DIR)/$(PACKAGE_NAME)
	@cp $(BUILD_DIR)/$(BINARY) $(DIST_DIR)/$(PACKAGE_NAME)/
	@cp README.md $(DIST_DIR)/$(PACKAGE_NAME)/ 2>/dev/null || true
	@cp LICENSE $(DIST_DIR)/$(PACKAGE_NAME)/ 2>/dev/null || true
ifeq ($(DETECTED_OS),windows)
	@cd $(DIST_DIR) && powershell Compress-Archive -Force -Path $(PACKAGE_NAME) -DestinationPath $(PACKAGE_NAME)-$(VERSION)-$(DETECTED_OS)-$(GOARCH).$(ARCHIVE_EXT)
	@echo "Package created: $(DIST_DIR)/$(PACKAGE_NAME)-$(VERSION)-$(DETECTED_OS)-$(GOARCH).$(ARCHIVE_EXT)"
else
	@cd $(DIST_DIR) && tar -czf $(PACKAGE_NAME)-$(VERSION)-$(DETECTED_OS)-$(GOARCH).$(ARCHIVE_EXT) $(PACKAGE_NAME)
	@echo "Package created: $(DIST_DIR)/$(PACKAGE_NAME)-$(VERSION)-$(DETECTED_OS)-$(GOARCH).$(ARCHIVE_EXT)"
endif
	@rm -rf $(DIST_DIR)/$(PACKAGE_NAME)

## release: Full release build and package
release: release-build package
	@echo "Release $(VERSION) ready for $(DETECTED_OS)-$(GOARCH)"
