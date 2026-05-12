#!/usr/bin/env bash
# Regenerates gRPC Go code from all .proto files into gen/go/
set -euo pipefail

OUT=gen/go

for proto in api/proto/**/**/*.proto; do
  protoc \
    --proto_path=api/proto \
    --go_out="$OUT" --go_opt=paths=source_relative \
    --go-grpc_out="$OUT" --go-grpc_opt=paths=source_relative \
    "$proto"
done

echo "proto-gen done"
