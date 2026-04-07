#!/usr/bin/env bash
# Regenerate pkg/docreaderpb from docreader.proto (requires protoc + protoc-gen-go + protoc-gen-go-grpc).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
export PATH="$(go env GOPATH)/bin:$PATH"
cd "$ROOT"
protoc -I pkg/docreaderpb \
  --go_out=pkg/docreaderpb --go_opt=paths=source_relative \
  --go-grpc_out=pkg/docreaderpb --go-grpc_opt=paths=source_relative \
  pkg/docreaderpb/docreader.proto
echo "ok: pkg/docreaderpb/*.pb.go"
