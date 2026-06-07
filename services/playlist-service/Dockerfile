# Build stage
FROM golang:alpine AS builder

WORKDIR /app

# Copy go mod và sum files
COPY go.mod ./
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o playlist-service cmd/server/main.go

# Production stage
FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/playlist-service .

EXPOSE 8086

CMD ["./playlist-service"]
