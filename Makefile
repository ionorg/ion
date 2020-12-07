GO_LDFLAGS = -ldflags "-s -w"
GO_VERSION = 1.14
GO_TESTPKGS:=$(shell go list ./... | grep -v cmd | grep -v conf | grep -v node)
GO_COVERPKGS:=$(shell echo $(GO_TESTPKGS) | paste -s -d ',')
TEST_UID:=$(shell id -u)
TEST_GID:=$(shell id -g)

all: build

clean:
	rm -rf bin

go_deps:
	go mod download
	go generate ./...

build: go_deps
	go build -o bin/biz $(GO_LDFLAGS) cmd/biz/json-rpc/main.go
	go build -o bin/islb $(GO_LDFLAGS) cmd/islb/main.go
	go build -o bin/sfu $(GO_LDFLAGS) cmd/sfu/main.go
	go build -o bin/avp $(GO_LDFLAGS) cmd/avp/main.go

start-services:
	docker network create ionnet || true
	docker-compose -f docker-compose.yml up -d redis nats etcd

stop-services:
	docker-compose -f docker-compose.yml stop redis nats etcd

run:
	docker-compose up --build

test: go_deps start-services
	go test \
		-timeout 120s \
		-coverpkg=${GO_COVERPKGS} -coverprofile=cover.out -covermode=atomic \
		-v -race ${GO_TESTPKGS}
protos:
	docker build -t protoc-builder ./pkg/grpc/biz && docker run -v $(CURDIR):/workspace protoc-builder protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative pkg/grpc/biz/biz.proto

