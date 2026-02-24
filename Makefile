# RDS Analyzer Makefile
# Build automation for the RDS analyzer

# Build variables
BINARY_NAME := rds-analyzer
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Container image variables
IMAGE_NAME ?= rds-analyzer
IMAGE_TAG ?= latest
IMAGE_PLATFORMS ?= linux/amd64,linux/arm64

# Autodetect container engine (podman first, then docker)
ifeq ($(origin ENGINE), undefined)
  ENGINE = podman
  ifeq ($(shell which $(ENGINE) 2>/dev/null),)
    ENGINE = docker
  endif
endif

# Go build flags
LDFLAGS := -ldflags "-X github.com/openshift-kni/rds-analyzer/internal/cli.Version=$(VERSION) \
                      -X github.com/openshift-kni/rds-analyzer/internal/cli.Commit=$(COMMIT) \
                      -X github.com/openshift-kni/rds-analyzer/internal/cli.BuildDate=$(BUILD_DATE)"

# Directories
BUILD_DIR := ./build
CMD_DIR := ./cmd/rds-analyzer

.PHONY: all build clean test lint install help fmt vet image-build image-push image-build-multiarch image-login

# Default target
all: build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for multiple platforms
build-all: build-linux build-darwin build-windows

build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)

build-darwin:
	@echo "Building for macOS..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)

build-windows:
	@echo "Building for Windows..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)

# Run tests
test:
	@echo "Running tests..."
	go test -v -race -cover ./...

# Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(BUILD_DIR)
	go test -v -race -coverprofile=$(BUILD_DIR)/coverage.out ./...
	go tool cover -html=$(BUILD_DIR)/coverage.out -o $(BUILD_DIR)/coverage.html
	@echo "Coverage report: $(BUILD_DIR)/coverage.html"

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install with:"; \
		echo "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	fi

# Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

# Install to GOBIN
install:
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) $(CMD_DIR)
	@echo "Installed to $(shell go env GOBIN)/$(BINARY_NAME)"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	go clean

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Verify dependencies
verify:
	@echo "Verifying dependencies..."
	go mod verify

# Show help
help:
	@echo "RDS Analyzer Build Targets:"
	@echo ""
	@echo "  make build                - Build the binary for current platform"
	@echo "  make build-all            - Build for Linux, macOS, and Windows"
	@echo "  make test                 - Run tests"
	@echo "  make test-coverage        - Run tests with coverage report"
	@echo "  make lint                 - Run golangci-lint"
	@echo "  make fmt                  - Format code"
	@echo "  make vet                  - Run go vet"
	@echo "  make install              - Install to GOBIN"
	@echo "  make clean                - Remove build artifacts"
	@echo "  make deps                 - Download and tidy dependencies"
	@echo "  make verify               - Verify dependencies"
	@echo "  make image-build          - Build container image for current platform"
	@echo "  make image-build-multiarch - Build multi-arch container image"
	@echo "  make image-push           - Push container image to registry"
	@echo "  make help                 - Show this help"
	@echo ""
	@echo "Container engine: $(ENGINE)"

# Build container image for current platform
image-build:
	@echo "Building container image with $(ENGINE)..."
	$(ENGINE) build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t $(IMAGE_NAME):$(IMAGE_TAG) .

# Build multi-arch container image
image-build-multiarch:
ifeq ($(ENGINE), podman)
	@echo "Building multi-arch container image with podman..."
	$(ENGINE) manifest rm $(IMAGE_NAME):$(IMAGE_TAG) 2>/dev/null || true
	$(ENGINE) manifest create $(IMAGE_NAME):$(IMAGE_TAG)
	@for arch in amd64 arm64; do \
		echo "Building for linux/$${arch}..."; \
		$(ENGINE) build \
			--platform linux/$${arch} \
			--build-arg VERSION=$(VERSION) \
			--build-arg COMMIT=$(COMMIT) \
			--build-arg BUILD_DATE=$(BUILD_DATE) \
			--manifest $(IMAGE_NAME):$(IMAGE_TAG) \
			. ; \
	done
else
	@echo "Building multi-arch container image with docker buildx..."
	$(ENGINE) buildx build \
		--platform $(IMAGE_PLATFORMS) \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t $(IMAGE_NAME):$(IMAGE_TAG) \
		--load .
endif

# Push container image to registry
image-push:
ifeq ($(ENGINE), podman)
	@echo "Pushing manifest with podman..."
	$(ENGINE) manifest push $(IMAGE_NAME):$(IMAGE_TAG) docker://$(IMAGE_NAME):$(IMAGE_TAG)
else
	@echo "Pushing image with docker..."
	$(ENGINE) push $(IMAGE_NAME):$(IMAGE_TAG)
endif

# Login to container registry
image-login:
	$(ENGINE) login $(IMAGE_REGISTRY)
