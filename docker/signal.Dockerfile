FROM golang:1.16-alpine as builder

WORKDIR /ion

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY pkg/ pkg/
COPY proto/ proto/
COPY cmd/ cmd/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o /ion/signal ./cmd/signal

FROM alpine

COPY --from=builder /ion/signal /signal

COPY configs/docker/signal.toml /configs/signal.toml

ENTRYPOINT ["/signal"]
CMD ["-c", "/configs/signal.toml"]
