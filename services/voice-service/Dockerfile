# Build stage
FROM golang:alpine AS builder

WORKDIR /app

# Copy go mod và sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy mã nguồn
COPY . .

# Biên dịch binary tĩnh (static binary)
RUN CGO_ENABLED=0 GOOS=linux go build -o voice-service cmd/server/main.go

# Production stage
FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/voice-service .

EXPOSE 8088

CMD ["./voice-service"]
