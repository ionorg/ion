FROM golang:1.16-alpine as builder

WORKDIR /ion

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY pkg/ pkg/
COPY proto/ proto/
COPY apps/ apps/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -installsuffix cgo -o /ion/app-room ./apps/room

FROM alpine

COPY --from=builder /ion/app-room /app-room

COPY configs/docker/app-room.toml /configs/app-room.toml

ENTRYPOINT ["/app-room"]
CMD ["-c", "/configs/app-room.toml"]
