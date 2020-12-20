
FROM golang:1.14.13-stretch as builder

RUN export GO111MODULE=on && go get github.com/golang/protobuf/protoc-gen-go@v1.3.2

RUN echo $GOPATH

FROM alpine:3.12.1

RUN apk --no-cache add protobuf

COPY --from=builder /go/bin/protoc-gen-go /usr/bin/

WORKDIR /workspace

CMD ["protoc"]
