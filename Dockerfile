# Build stage
FROM golang:1.21-alpine AS builder

# Install git and ca-certificates
RUN apk --no-cache add git ca-certificates

# Set working directory
WORKDIR /app

# Copy go modules first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o imap-backup .

# Final stage
FROM alpine:latest

# Install ca-certificates for SSL/TLS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN adduser -D -s /bin/sh imap-backup

# Set working directory
WORKDIR /home/imap-backup

# Copy the binary from builder stage
COPY --from=builder /app/imap-backup .

# Change ownership to non-root user
RUN chown imap-backup:imap-backup ./imap-backup

# Switch to non-root user
USER imap-backup

# Expose volume for backups
VOLUME ["/home/imap-backup/backup"]

# Entry point
ENTRYPOINT ["./imap-backup"]
CMD ["--help"]