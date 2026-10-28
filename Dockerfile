# Stage 1: Build the statically linked Go binary
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install git and ca-certificates (needed for SSL calls to Gemini)
RUN apk add --no-cache git ca-certificates

# Cache dependencies
COPY go.mod ./
RUN go mod download

# Copy source code
COPY . .

# Build statically compiled binary with debugging symbols stripped to minimize size
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/commitlint-cli ./cmd/cli

# Stage 2: Lightweight runtime image (<20MB)
FROM alpine:3.19

WORKDIR /app

# Copy ca-certificates for outbound TLS connections to Google APIs
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/server /app/server
COPY --from=builder /app/commitlint-cli /app/commitlint-cli

# Run as non-root user for security best practices
RUN adduser -D -u 10001 appuser
USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/server"]

