# IMAP Backup Tool Makefile

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=imap-backup

# Build info
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT ?= $(shell git rev-parse HEAD)
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Linker flags
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

.PHONY: build build-all clean test deps install help coverage lint staticcheck security sonar quality ci

# Build the application
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)

# Build for multiple platforms
build-all:
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME) $(BINARY_NAME)-*
	rm -f coverage.out coverage.html
	rm -f *.xml *.json *.out

# Run tests
test:
	$(GOTEST) -v -race -timeout 30s ./...

# Run tests with coverage
coverage:
	$(GOTEST) -v -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Install dependencies
deps:
	$(GOMOD) tidy
	$(GOMOD) download

# Install development tools
install-tools:
	$(GOGET) -u github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GOGET) -u honnef.co/go/tools/cmd/staticcheck@latest
	$(GOGET) -u github.com/securecodewarrior/gosec/v2/cmd/gosec@latest

# Run linters
lint:
	golangci-lint run ./...

# Run staticcheck
staticcheck:
	staticcheck ./...

# Run security analysis
security:
	gosec ./...

# SonarQube integration
sonar-prepare: coverage
	@echo "Preparing reports for SonarQube..."
	golangci-lint run --out-format checkstyle ./... > golangci-lint-report.xml || true
	$(GOCMD) vet ./... 2> govet-report.out || true
	gosec -fmt sonarqube -out gosec-report.json ./... || true

sonar: sonar-prepare
	@echo "Running SonarQube analysis..."
	sonar-scanner \
		-Dsonar.projectKey=imap-backup \
		-Dsonar.sources=. \
		-Dsonar.exclusions=**/*_test.go,**/vendor/**,**/testdata/**,**/backup/**,**/test-** \
		-Dsonar.tests=. \
		-Dsonar.test.inclusions=**/*_test.go \
		-Dsonar.go.coverage.reportPaths=coverage.out \
		-Dsonar.go.golangci-lint.reportPaths=golangci-lint-report.xml \
		-Dsonar.go.govet.reportPaths=govet-report.out \
		-Dsonar.go.gosec.reportPaths=gosec-report.json

# Docker support for SonarQube
sonar-docker: sonar-prepare
	@echo "Running SonarQube analysis using Docker..."
	docker run --rm \
		-v $(PWD):/usr/src \
		-e SONAR_HOST_URL=$$SONAR_HOST_URL \
		-e SONAR_LOGIN=$$SONAR_TOKEN \
		sonarsource/sonar-scanner-cli:latest

# Quality checks
quality: lint staticcheck security test coverage

# CI/CD targets
ci: deps quality build

# Install locally
install:
	$(GOCMD) install

# Show help
help:
	@echo "Available targets:"
	@echo "  build         - Build the application"
	@echo "  build-all     - Build for multiple platforms"
	@echo "  clean         - Clean build artifacts"
	@echo "  test          - Run tests"
	@echo "  coverage      - Run tests with coverage report"
	@echo "  deps          - Install dependencies"
	@echo "  install-tools - Install development tools"
	@echo "  lint          - Run golangci-lint"
	@echo "  staticcheck   - Run staticcheck"
	@echo "  security      - Run gosec security scanner"
	@echo "  quality       - Run all quality checks"
	@echo "  sonar-prepare - Prepare reports for SonarQube"
	@echo "  sonar         - Run SonarQube analysis"
	@echo "  sonar-docker  - Run SonarQube analysis using Docker"
	@echo "  ci            - Full CI pipeline (deps, quality, build)"
	@echo "  install       - Install locally"
	@echo "  help          - Show this help message"

# Default target
all: deps build