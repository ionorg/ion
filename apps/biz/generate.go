package main

// ============================================================================
// GO
// ============================================================================
// GRPC & Protobuf
//go:generate /usr/bin/env bash -c "echo 'Generating [biz] protobuf and grpc services for Go, outdir=$OUTDIR'"
//go:generate /usr/bin/env bash -c "if command -v protoc &>/dev/null; then protoc ./proto/biz.proto -I./proto -I../../ --go_opt=module=github.com/pion/ion --go_out=../../$OUTDIR --go-grpc_opt=module=github.com/pion/ion --go-grpc_out=../../$OUTDIR; fi"
