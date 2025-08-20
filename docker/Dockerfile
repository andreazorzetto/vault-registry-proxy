# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o vault-docker-proxy .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/vault-docker-proxy .

# Make the binary executable and set proper ownership for nobody user (65534)
RUN chmod +x vault-docker-proxy && \
    chown -R nobody:nobody /app

# Switch to nobody user (UID 65534)
USER nobody

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./vault-docker-proxy"]