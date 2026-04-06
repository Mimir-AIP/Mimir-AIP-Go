# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy Go module metadata and frontend server sources
COPY go.mod go.sum ./
COPY cmd/frontend ./cmd/frontend
COPY frontend ./frontend

# Build the frontend server
RUN go build -o /app/frontend-server ./cmd/frontend

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary and full static asset tree
COPY --from=builder /app/frontend-server .
COPY --from=builder /app/frontend/index.html .
COPY --from=builder /app/frontend/styles.css .
COPY --from=builder /app/frontend/app.js .
COPY --from=builder /app/frontend/lib ./lib
COPY --from=builder /app/frontend/context ./context
COPY --from=builder /app/frontend/hooks ./hooks
COPY --from=builder /app/frontend/components ./components
COPY --from=builder /app/frontend/pages ./pages
COPY --from=builder /app/frontend/vendor ./vendor

# Expose port
EXPOSE 3000

# Run the frontend server
CMD ["./frontend-server"]
