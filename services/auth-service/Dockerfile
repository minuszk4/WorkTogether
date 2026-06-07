# Build stage
FROM golang:alpine AS builder

WORKDIR /app

# Copy go mod và sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy mã nguồn
COPY . .

# Biên dịch binary tĩnh (static binary)
RUN CGO_ENABLED=0 GOOS=linux go build -o auth-service cmd/server/main.go

# Production stage
FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/auth-service .

EXPOSE 8081

CMD ["./auth-service"]
