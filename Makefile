GO_LDFLAGS = -ldflags "-s -w"
GO_VERSION = 1.14
GO_TESTPKGS:=$(shell go list ./... | grep -v cmd | grep -v conf | grep -v node)
GO_COVERPKGS:=$(shell echo $(GO_TESTPKGS) | paste -s -d ',')
TEST_UID:=$(shell id -u)
TEST_GID:=$(shell id -g)

all: nodes

deps:
	./scripts/install-run-deps.sh

go_deps:
	go mod download

clean:
	rm -rf bin

upx:
	upx -9 bin/*

example:
	go build -o bin/service-node $(GO_LDFLAGS) examples/service/service-node.go
	go build -o bin/service-watch $(GO_LDFLAGS) examples/watch/service-watch.go

nodes: go_deps
	go build -o bin/biz $(GO_LDFLAGS) cmd/biz/main.go
	go build -o bin/islb $(GO_LDFLAGS) cmd/islb/main.go
	go build -o bin/sfu $(GO_LDFLAGS) cmd/sfu/main.go

start_test_services:
	docker network create ionnet || true
	docker-compose -f docker-compose.stable.yml up -d redis nats etcd

test: nodes start_test_services
	go test \
		-coverpkg=${GO_COVERPKGS} -coverprofile=cover.out -covermode=atomic \
		-v -race ${GO_TESTPKGS}
