FROM golang:1.14.13-stretch

ENV GO111MODULE=on

WORKDIR $GOPATH/src/github.com/pion/ion

COPY go.mod go.sum ./
RUN cd $GOPATH/src/github.com/pion/ion && go mod download

COPY pkg/ $GOPATH/src/github.com/pion/ion/pkg
COPY cmd/ $GOPATH/src/github.com/pion/ion/cmd
COPY proto/ $GOPATH/src/github.com/pion/ion/proto

WORKDIR $GOPATH/src/github.com/pion/ion/cmd/signal
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /sig .

FROM alpine:3.12.1

RUN apk --no-cache add ca-certificates
COPY --from=0 /sig /usr/local/bin/sig

COPY configs/docker/sig.toml /configs/sig.toml

ENTRYPOINT ["/usr/local/bin/sig"]
CMD ["-c", "/configs/sig.toml"]
