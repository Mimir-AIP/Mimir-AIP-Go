# Multi-stage build for minimal, secure Go server image
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN apk add --no-cache git && go mod tidy && go build -o mimir-aip-server main.go

# Use distroless for minimal runtime image
FROM gcr.io/distroless/base-debian11
WORKDIR /app
COPY --from=builder /app/mimir-aip-server /app/mimir-aip-server
COPY --from=builder /app/config.yaml /app/config.yaml
EXPOSE 8080
USER nonroot
ENTRYPOINT ["/app/mimir-aip-server"]
