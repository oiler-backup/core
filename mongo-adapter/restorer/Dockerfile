# Stage 1: Build the binary
FROM golang:1.24-alpine AS builder

# Install dependencies
# Set up working directory
WORKDIR /app

# Copy source code
COPY . .

# Build the application
RUN go build -o backup-restore-app .

# Stage 2: Run the application
FROM alpine:latest

# Install runtime dependencies
RUN apk add mongodb-tools

# Copy the binary from the builder stage
COPY --from=builder /app/backup-restore-app /usr/local/bin/backup-restore-app

# Set entrypoint
ENTRYPOINT ["backup-restore-app"]