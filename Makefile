GO_LDFLAGS = -ldflags "-s -w"
GO_VERSION = 1.14
GO_TESTPKGS:=$(shell go list ./... | grep -v cmd | grep -v conf | grep -v node | grep -v util)

all: build

clean:
	rm -rf bin

go_deps:
	go mod download
	go generate ./...

build: go_deps
	go build -o bin/biz $(GO_LDFLAGS) cmd/biz/main.go
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

test:
	go test \
		-timeout 120s \
		-coverprofile=cover.out -covermode=atomic \
		-v -race ${GO_TESTPKGS} 
