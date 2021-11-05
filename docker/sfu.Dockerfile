FROM golang:1.16-alpine as builder

WORKDIR /ion

COPY go.mod go.mod
COPY go.sum go.sum

COPY pkg/ pkg/
COPY proto/ proto/
COPY cmd/ cmd/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o /ion/sfu ./cmd/sfu

FROM alpine

COPY --from=builder /ion/sfu /sfu

COPY configs/docker/sfu.toml /configs/sfu.toml

ENTRYPOINT ["/sfu"]
CMD ["-c", "/configs/sfu.toml"]
