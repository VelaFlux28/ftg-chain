# FTG Energy Chain — Build System
# US Patent 11,962,710

BINARY_NAME = ftgd
VERSION = 0.1.0
CHAIN_ID = ftg-energy-1
DOCKER_IMAGE = ftg-energy
GO = go
GOFLAGS = -v

.PHONY: all build install clean test docker-build docker-run init start

all: build

# Build the ftgd binary
build:
	@echo "Building $(BINARY_NAME) v$(VERSION)..."
	$(GO) build $(GOFLAGS) -o build/$(BINARY_NAME) ./cmd/ftgd/
	@echo "Binary: build/$(BINARY_NAME)"

# Install to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	$(GO) install $(GOFLAGS) ./cmd/ftgd/

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf build/
	$(GO) clean

# Run tests
test:
	@echo "Running tests..."
	$(GO) test ./... -v -count=1

# Build Docker image
docker-build:
	@echo "Building Docker image: $(DOCKER_IMAGE):$(VERSION)..."
	docker build -t $(DOCKER_IMAGE):$(VERSION) -t $(DOCKER_IMAGE):latest .

# Run testnet via Docker Compose
docker-run:
	@echo "Starting FTG Energy testnet..."
	docker compose up -d
	@echo "Testnet running. RPC: http://localhost:26657"

# Stop testnet
docker-stop:
	docker compose down

# Initialize a new node
init:
	@echo "Initializing FTG Energy node..."
	build/$(BINARY_NAME) init ftg-genesis-validator

# Start the node
start:
	@echo "Starting FTG Energy node..."
	build/$(BINARY_NAME) start

# Show version
version:
	@build/$(BINARY_NAME) version

# Genesis operations
genesis-validate:
	@build/$(BINARY_NAME) genesis validate

# Treasury overview
treasury:
	@build/$(BINARY_NAME) treasury overview

# Lint
lint:
	@echo "Linting..."
	golangci-lint run ./...

# Format
fmt:
	@echo "Formatting..."
	$(GO) fmt ./...

# Help
help:
	@echo "FTG Energy Chain Build System"
	@echo ""
	@echo "Commands:"
	@echo "  make build          Build the ftgd binary"
	@echo "  make install        Install to GOPATH/bin"
	@echo "  make clean          Clean build artifacts"
	@echo "  make test           Run all tests"
	@echo "  make docker-build   Build Docker image"
	@echo "  make docker-run     Start testnet (Docker Compose)"
	@echo "  make docker-stop    Stop testnet"
	@echo "  make init           Initialize a new node"
	@echo "  make start          Start the node"
	@echo "  make version        Show version"
	@echo "  make treasury       Show treasury overview"
	@echo "  make lint           Run linter"
	@echo "  make fmt            Format code"
