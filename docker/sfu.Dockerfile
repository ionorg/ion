FROM golang:1.14.4-stretch

ENV GO111MODULE=on

WORKDIR $GOPATH/src/github.com/sssgun/ion/ion

COPY go.mod go.sum ./
RUN cd $GOPATH/src/github.com/sssgun/ion/ion && go mod download

COPY pkg/ $GOPATH/src/github.com/sssgun/ion/ion/pkg
COPY cmd/ $GOPATH/src/github.com/sssgun/ion/ion/cmd

WORKDIR $GOPATH/src/github.com/sssgun/ion/ion/cmd/sfu
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /sfu .

FROM alpine:3.12.0

RUN apk --no-cache add ca-certificates
COPY --from=0 /sfu /usr/local/bin/sfu

COPY configs/docker/sfu.toml /configs/sfu.toml

ENTRYPOINT ["/usr/local/bin/sfu"]
CMD ["-c", "/configs/sfu.toml"]
