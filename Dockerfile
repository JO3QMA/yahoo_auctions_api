# Build stage
FROM golang:1.25.4-alpine AS builder

WORKDIR /app

# Install ca-certificates for HTTPS requests during build
RUN apk add --no-cache ca-certificates

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o yahoo-auctions-server \
    ./cmd/server

# Runtime stage
FROM gcr.io/distroless/base:latest

# Copy the binary from builder stage
COPY --from=builder /app/yahoo-auctions-server /usr/local/bin/yahoo-auctions-server

# Expose port and set default port
EXPOSE 8080
ENV PORT=8080

# Run the binary
CMD ["yahoo-auctions-server"]
