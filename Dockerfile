# Multi-stage production build

# Stage 1: Frontend build
FROM node:22-alpine AS frontend-builder

WORKDIR /app/web

COPY web/package*.json ./
RUN npm ci

COPY web .
RUN npm run build

# Stage 2: Backend build
FROM golang:1.25-alpine AS backend-builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

# Copy Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY cmd ./cmd
COPY internal ./internal
COPY api ./api
COPY gen ./gen

# Build the server and public CLI binaries
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s" \
    -o registry ./cmd/registry
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s" \
    -o alauda ./cmd/registryctl

# Stage 3: Runtime
FROM alpine:3.18

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates sqlite

# Create non-root user
RUN addgroup -g 1000 registry && \
    adduser -D -u 1000 -G registry registry

# Copy server and canonical CLI binaries from builder
COPY --from=backend-builder /app/registry /usr/local/bin/registry
COPY --from=backend-builder /app/alauda /usr/local/bin/alauda
RUN ln -s /usr/local/bin/alauda /usr/local/bin/registryctl

# Copy static files (embedded in binary, but also available)
COPY --from=frontend-builder /app/web/dist ./web/dist

# Create data directory
RUN mkdir -p /data && chown -R registry:registry /data

USER registry

EXPOSE 9700

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget -qO- http://127.0.0.1:9700/healthz || exit 1

ENTRYPOINT ["/usr/local/bin/registry"]
CMD ["server"]
