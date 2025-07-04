# IMAP Backup Tool Makefile

.PHONY: build clean test install help

# Build the application
build:
	go build -o imap-backup

# Build for multiple platforms
build-all:
	GOOS=darwin GOARCH=amd64 go build -o imap-backup-darwin-amd64
	GOOS=darwin GOARCH=arm64 go build -o imap-backup-darwin-arm64
	GOOS=linux GOARCH=amd64 go build -o imap-backup-linux-amd64
	GOOS=windows GOARCH=amd64 go build -o imap-backup-windows-amd64.exe

# Clean build artifacts
clean:
	rm -f imap-backup imap-backup-*

# Run tests
test:
	go test ./...

# Install dependencies
deps:
	go mod tidy
	go mod download

# Install locally
install:
	go install

# Show help
help:
	@echo "Available targets:"
	@echo "  build     - Build the application"
	@echo "  build-all - Build for multiple platforms"
	@echo "  clean     - Clean build artifacts"
	@echo "  test      - Run tests"
	@echo "  deps      - Install dependencies"
	@echo "  install   - Install locally"
	@echo "  help      - Show this help message"

# Default target
all: deps build