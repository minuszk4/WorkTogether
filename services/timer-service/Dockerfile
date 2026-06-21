# Build stage
FROM golang:alpine AS builder

WORKDIR /app

# Copy go mod
COPY go.mod ./
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o timer-service cmd/server/main.go

# Production stage
FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/timer-service .

EXPOSE 8093

CMD ["./timer-service"]
