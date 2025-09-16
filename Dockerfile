# Build stage
FROM golang:1.22-alpine AS builder

# Set build arguments
ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags="-w -s -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
    -o helios \
    ./cmd/helios-cli

# Runtime stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk --no-cache add ca-certificates git

# Create non-root user
RUN adduser -D -s /bin/sh helios

# Set working directory
WORKDIR /home/helios

# Copy binary from builder
COPY --from=builder /app/helios /usr/local/bin/helios

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Switch to non-root user
USER helios

# Expose default port (if applicable)
EXPOSE 8080

# Set entrypoint
ENTRYPOINT ["helios"]

# Default command
CMD ["--help"]

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD helios version || exit 1

# Labels
LABEL org.opencontainers.image.title="Helios CLI"
LABEL org.opencontainers.image.description="Fast Version Control for AI Agents"
LABEL org.opencontainers.image.url="https://github.com/good-night-oppie/helios"
LABEL org.opencontainers.image.source="https://github.com/good-night-oppie/helios"
LABEL org.opencontainers.image.documentation="https://github.com/good-night-oppie/helios/blob/main/README.md"
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.vendor="Oppie Thunder"