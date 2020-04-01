FROM golang:1.13.7-stretch

ENV GO111MODULE=on

WORKDIR $GOPATH/src/github.com/pion/ion

RUN echo $GOPATH/src/github.com/pion/ion

COPY go.mod go.sum ./
RUN cd $GOPATH/src/github.com/pion/ion && go mod download

COPY . $GOPATH/src/github.com/pion/ion

WORKDIR $GOPATH/src/github.com/pion/ion/cmd/avp
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /avp .

FROM alpine:3.9.5

RUN apk --no-cache add ca-certificates
COPY --from=0 /avp /usr/local/bin/avp

ENTRYPOINT ["/usr/local/bin/avp"]
