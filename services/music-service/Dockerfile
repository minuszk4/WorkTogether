# Build stage
FROM golang:alpine AS builder

WORKDIR /app

# Copy go mod và sum files
COPY go.mod ./
# Note: we run tidy later to create go.sum
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o music-service cmd/server/main.go

# Production stage
FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/music-service .

EXPOSE 8085

CMD ["./music-service"]
