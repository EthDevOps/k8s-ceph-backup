.PHONY: build clean test install deps help

BINARY_NAME=k8s-ceph-backup
BINARY_PATH=./$(BINARY_NAME)
GO_FILES=$(shell find . -name "*.go" -type f)

# Default target
all: build

# Build the binary
build: deps
	@echo "Building $(BINARY_NAME)..."
	go build -ldflags="-s -w" -o $(BINARY_NAME)
	@echo "Binary built: $(BINARY_PATH)"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME)
	go clean

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Install binary to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME) to $(GOPATH)/bin..."
	go install

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	golangci-lint run

# Run the application with default settings
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME)

# Build for multiple platforms
build-all: deps
	@echo "Building for multiple platforms..."
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY_NAME)-linux-amd64
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY_NAME)-darwin-amd64
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY_NAME)-windows-amd64.exe
	@echo "Built binaries for Linux, macOS, and Windows"

# Show help
help:
	@echo "Available targets:"
	@echo "  build      - Build the binary"
	@echo "  deps       - Install dependencies"
	@echo "  clean      - Clean build artifacts"
	@echo "  test       - Run tests"
	@echo "  install    - Install binary to GOPATH/bin"
	@echo "  fmt        - Format code"
	@echo "  lint       - Lint code"
	@echo "  run        - Build and run the application"
	@echo "  build-all  - Build for multiple platforms"
	@echo "  help       - Show this help message"