FROM golang:1.14.4-stretch

ENV GO111MODULE=on

WORKDIR $GOPATH/src/github.com/sssgun/ion/ion

COPY go.mod go.sum ./
RUN cd $GOPATH/src/github.com/sssgun/ion/ion && go mod download

COPY pkg/ $GOPATH/src/github.com/sssgun/ion/ion/pkg
COPY cmd/ $GOPATH/src/github.com/sssgun/ion/ion/cmd

WORKDIR $GOPATH/src/github.com/sssgun/ion/ion/cmd/biz
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /biz .

FROM alpine:3.12.0

RUN apk --no-cache add ca-certificates
COPY --from=0 /biz /usr/local/bin/biz

COPY configs/docker/biz.toml /configs/biz.toml

ENTRYPOINT ["/usr/local/bin/biz"]
CMD ["-c", "/configs/biz.toml"]
