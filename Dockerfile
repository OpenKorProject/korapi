# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Cache dependency downloads
COPY go.mod go.sum ./
RUN go mod download

# Build application
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o korapi ./cmd/korapi

# Final stage
FROM alpine:3.20

RUN apk add --no-cache curl

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/korapi .

# Health check
HEALTHCHECK --interval=5s --timeout=3s --retries=5 \
  CMD curl -f http://localhost:8080/healthz || exit 1

EXPOSE 8080

CMD ["./korapi"]
