# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy root go.work files
COPY go.work go.work.sum ./

# Copy shared pkg
COPY pkg ./pkg

# Copy target service
COPY services/notification-service ./services/notification-service

WORKDIR /app/services/notification-service
RUN go mod download

# Biên dịch binary tĩnh (static binary)
RUN CGO_ENABLED=0 GOOS=linux go build -o notification-service cmd/server/main.go

# Production stage
FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/services/notification-service/notification-service .

EXPOSE 8089

CMD ["./notification-service"]
