# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy root go.work files
COPY go.work go.work.sum ./

# Copy shared pkg
COPY pkg ./pkg

# Copy workspace services so go.work can resolve all local modules
COPY services ./services

WORKDIR /app/services/user-service
RUN go mod download

# Biên dịch binary tĩnh (static binary)
RUN CGO_ENABLED=0 GOOS=linux go build -o user-service cmd/server/main.go

# Production stage
FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata && addgroup -S app && adduser -S app -G app

COPY --from=builder --chown=app:app /app/services/user-service/user-service .

EXPOSE 8082

USER app

CMD ["./user-service"]
