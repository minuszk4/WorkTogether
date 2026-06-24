# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy root go.work files
COPY go.work go.work.sum ./

# Copy shared pkg
COPY pkg ./pkg

# Copy target service
COPY services/voice-service ./services/voice-service

WORKDIR /app/services/voice-service
RUN go mod download

# Biên dịch binary tĩnh (static binary)
RUN CGO_ENABLED=0 GOOS=linux go build -o voice-service cmd/server/main.go

# Production stage
FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/services/voice-service/voice-service .

EXPOSE 8088

CMD ["./voice-service"]
