FROM golang:1.13.7-stretch

ENV GO111MODULE=on

WORKDIR $GOPATH/src/github.com/pion/ion

COPY go.mod go.sum ./
RUN cd $GOPATH/src/github.com/pion/ion && go mod download

COPY . $GOPATH/src/github.com/pion/ion

WORKDIR $GOPATH/src/github.com/pion/ion/cmd/ion
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /ion .

FROM alpine:3.9.5
RUN apk --no-cache add ca-certificates
COPY --from=0 /ion /usr/local/bin/ion

ADD https://raw.githubusercontent.com/Eficode/wait-for/master/wait-for /wait-for
RUN chmod +x /wait-for

ENTRYPOINT ["/usr/local/bin/ion"]