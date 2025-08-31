# Build stage
FROM golang:1.22-alpine AS builder

# Install git and ca-certificates (needed for downloading dependencies)
RUN apk update && apk add --no-cache git ca-certificates tzdata && update-ca-certificates

# Create appuser for security
RUN adduser -D -g '' appuser

# Set working directory
WORKDIR /build

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o main cmd/api/main.go

# Final stage
FROM scratch

# Import from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/passwd /etc/passwd

# Copy the binary
COPY --from=builder /build/main /app/main

# Create uploads directory
COPY --from=builder --chown=appuser:appuser /tmp /app/uploads

# Use an unprivileged user
USER appuser

# Expose port
EXPOSE 8080

# Set working directory
WORKDIR /app

# Run the binary
ENTRYPOINT ["./main"]
