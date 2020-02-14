FROM golang:1.13.7-stretch

ENV GO111MODULE=on

COPY . $GOPATH/src/github.com/pion/ion
RUN cd $GOPATH/src/github.com/pion/ion && go mod download

WORKDIR $GOPATH/src/github.com/pion/ion
RUN cd ./cmd/ion && go build

ENTRYPOINT ["./cmd/ion/ion"]

