# Production-style image for cmd/notex (unified Notex API + LangGraph).
# Build from repository root: docker build -f docker/notex/Dockerfile -t youmind/notex:local .
FROM golang:1.25-alpine AS builder
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/notex ./cmd/notex

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata curl
WORKDIR /app
COPY --from=builder /out/notex /app/notex
# Default config when no bind-mount (override with volume in compose).
COPY config.example.yaml /app/config.yaml
ENV NOTEX_DATA_ROOT=/data/notex \
    NOTEX_PORT=:8787 \
    DEERFLOW_CONFIG_PATH=/app/config.yaml
RUN mkdir -p /data/notex
EXPOSE 8787
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
  CMD curl -fsS http://127.0.0.1:8787/health >/dev/null || exit 1
ENTRYPOINT ["/app/notex"]
