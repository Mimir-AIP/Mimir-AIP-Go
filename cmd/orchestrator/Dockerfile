# Build stage
FROM golang:1.25 AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the orchestrator binary with CGO enabled (required for Go plugins)
# Use -trimpath to ensure reproducible builds for plugin compatibility
RUN CGO_ENABLED=1 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app/orchestrator ./cmd/orchestrator

# Runtime stage — based on the full Go image so that dynamic storage plugins
# can be compiled at runtime via go build -buildmode=plugin.
# The Go toolchain, gcc (via debian base), and git are all required.
FROM golang:1.25

# Install git for cloning plugin repositories
RUN apt-get update && \
    apt-get install -y --no-install-recommends git && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the pre-built orchestrator binary
COPY --from=builder /app/orchestrator .

# Copy the host module files and source — the dynamic plugin loader compiles
# external plugins inside /app using these files, so the plugin's code is built
# against the exact same module the orchestrator was compiled from.
COPY --from=builder /app/go.mod /app/go.sum ./
COPY --from=builder /app/pkg ./pkg
COPY --from=builder /app/cmd ./cmd

# Pre-warm the module cache so the first plugin install does not require a
# full go mod download. This does not add new dependencies.
RUN go mod download

# Directories for compiled .so cache and temporary git clones
RUN mkdir -p /app/storage-plugins /tmp/sp-temp

EXPOSE 8080

CMD ["./orchestrator"]
