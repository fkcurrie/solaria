# Multi-Stage Ultra-Compact Go Container for Cloud Run (~12MB)
FROM golang:alpine AS builder

WORKDIR /build

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
COPY . .

# Compile static Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /solaria-server ./cmd/cloud-server

# Final Runtime Image
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /solaria-server /app/solaria-server

ENV PORT=8080
EXPOSE 8080

ENTRYPOINT ["/app/solaria-server"]
