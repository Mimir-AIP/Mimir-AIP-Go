# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy frontend files
COPY frontend/ ./

# Build the frontend server
RUN go build -o /app/frontend-server server.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary and static files
COPY --from=builder /app/frontend-server .
COPY frontend/index.html frontend/styles.css frontend/app.js ./

# Expose port
EXPOSE 3000

# Run the frontend server
CMD ["./frontend-server"]
