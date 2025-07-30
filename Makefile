.PHONY: build run clean install test lint fmt help

# Binary name
BINARY_NAME=termidar
BINARY_PATH=./bin/$(BINARY_NAME)

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

# Build flags
LDFLAGS=-ldflags "-s -w"

# Default target
all: build

## help: Show this help message
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	@$(GOBUILD) $(LDFLAGS) -o $(BINARY_PATH) -v

## run: Run the application
run:
	@$(GOCMD) run main.go

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	@$(GOCLEAN)
	@rm -rf bin/

## install: Install the binary to $GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	@$(GOCMD) install $(LDFLAGS)

## test: Run tests
test:
	@echo "Running tests..."
	@$(GOTEST) -v ./...

## fmt: Format code
fmt:
	@echo "Formatting code..."
	@$(GOFMT) -s -w .

## lint: Run linter
lint:
	@echo "Running linter..."
	@$(GOLINT) run

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	@$(GOMOD) download

## update: Update dependencies
update:
	@echo "Updating dependencies..."
	@$(GOMOD) tidy

## build-all: Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p bin
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-amd64.exe