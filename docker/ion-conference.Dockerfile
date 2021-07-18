FROM golang:1.14.13-stretch

ENV GO111MODULE=on

WORKDIR $GOPATH/src/github.com/pion/ion

COPY go.mod go.sum ./
RUN cd $GOPATH/src/github.com/pion/ion && go mod download

COPY pkg/ $GOPATH/src/github.com/pion/ion/pkg
COPY proto/ $GOPATH/src/github.com/pion/ion/proto
COPY apps $GOPATH/src/github.com/pion/ion/apps

WORKDIR $GOPATH/src/github.com/pion/ion/apps/ion-conference
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /ion-conference .

FROM alpine:3.12.1

RUN apk --no-cache add ca-certificates
COPY --from=0 /app-biz /usr/local/bin/ion-conference

COPY configs/docker/app-biz.toml /configs/app-biz.toml
COPY configs/docker/sfu.toml /configs/sfu.toml

ENTRYPOINT ["/usr/local/bin/ioin-conference"]
CMD ["-bc", "/configs/app-biz.toml", "-sc", "/configs/sfu.toml"]
