FROM golang:1.16-alpine as builder

WORKDIR /ion

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY pkg/ pkg/
COPY proto/ proto/
COPY cmd/ cmd/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o /ion/islb ./cmd/islb

FROM alpine

COPY --from=builder /ion/islb /islb

COPY configs/docker/islb.toml /configs/islb.toml

ENTRYPOINT ["/islb"]
CMD ["-c", "/configs/islb.toml"]
