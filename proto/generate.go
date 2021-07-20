package main

// ============================================================================
// GO
// ============================================================================
// GRPC & Protobuf
//go:generate /usr/bin/env bash -c "echo 'Generating [ion] protobuf and grpc services for Go, outdir=$OUTDIR'"
//go:generate protoc ./ion.proto -I../ -I./ --go_opt=module=github.com/pion/ion --go_out=../$OUTDIR
//go:generate protoc ./debug.proto -I../ -I./ --go_opt=module=github.com/pion/ion --go_out=../$OUTDIR --experimental_allow_proto3_optional
//go:generate protoc ./islb.proto -I../ -I./ --go_opt=module=github.com/pion/ion --go_out=../$OUTDIR --go-grpc_opt=module=github.com/pion/ion --go-grpc_out=../$OUTDIR
//go:generate protoc ./rtc.proto -I../ -I./ --go_opt=module=github.com/pion/ion --go_out=../$OUTDIR --go-grpc_opt=module=github.com/pion/ion --go-grpc_out=../$OUTDIR
