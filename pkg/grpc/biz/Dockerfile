
FROM golang:1.14.13-stretch as builder

RUN go get google.golang.org/grpc/cmd/protoc-gen-go-grpc
RUN go get google.golang.org/protobuf/cmd/protoc-gen-go

RUN echo $GOPATH

FROM alpine:3.12.1

RUN apk --no-cache add protobuf

COPY --from=builder /go/bin/protoc-gen-go-grpc /usr/bin/
COPY --from=builder /go/bin/protoc-gen-go /usr/bin/

WORKDIR /workspace

CMD ["protoc"]
