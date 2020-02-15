GO_LDFLAGS = -ldflags "-s -w"

all: example nodes

deps:
	./scripts/install-run-deps.sh

clean:
	rm -rf bin

upx:
	upx -9 bin/*

example:
	go build -o bin/service-node $(GO_LDFLAGS) examples/service/service-node.go
	go build -o bin/service-watch $(GO_LDFLAGS) examples/watch/service-watch.go

nodes:
	go build -o bin/node-ion $(GO_LDFLAGS) cmd/ion/main.go
	go build -o bin/node-islb $(GO_LDFLAGS) cmd/islb/main.go
	go build -o bin/node-sfu $(GO_LDFLAGS) cmd/sfu/main.go
