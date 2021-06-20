GO_LDFLAGS = -ldflags "-s -w"
GO_VERSION = 1.14
GO_TESTPKGS:=$(shell go list ./... | grep -v cmd | grep -v conf | grep -v node)
GO_COVERPKGS:=$(shell echo $(GO_TESTPKGS) | paste -s -d ',')
TEST_UID:=$(shell id -u)
TEST_GID:=$(shell id -g)

all: core app

go_deps:
	go mod download

core: go_deps
	go build -o bin/islb $(GO_LDFLAGS) cmd/islb/main.go
	go build -o bin/sfu $(GO_LDFLAGS) cmd/sfu/main.go
	go build -o bin/avp $(GO_LDFLAGS) cmd/avp/main.go
	go build -o bin/signal $(GO_LDFLAGS) cmd/signal/main.go

app:
	go build -o bin/app-biz $(GO_LDFLAGS) apps/biz/main.go

clean:
	rm -rf bin

start-bin:

start-services:
	docker network create ionnet || true
	docker-compose -f docker-compose.yml up -d redis nats

stop-services:
	docker-compose -f docker-compose.yml stop redis nats

run:
	docker-compose up --build

test: go_deps start-services
	go test \
		-timeout 120s \
		-coverpkg=${GO_COVERPKGS} -coverprofile=cover.out -covermode=atomic \
		-v -race ${GO_TESTPKGS}

proto-gen-from-docker:
	docker build -t go-protoc ./proto
	docker run -v $(CURDIR):/workspace go-protoc proto

proto: proto_core proto_app

proto_core: 
	protoc proto/ion/ion.proto --go_opt=module=github.com/pion/ion --go_out=. --go-grpc_opt=module=github.com/pion/ion --go-grpc_out=.
	protoc proto/debug/debug.proto --experimental_allow_proto3_optional --go_opt=module=github.com/pion/ion --go_out=. --go-grpc_opt=module=github.com/pion/ion --go-grpc_out=.
	protoc proto/sfu/sfu.proto --go_opt=module=github.com/pion/ion --go_out=. --go-grpc_opt=module=github.com/pion/ion --go-grpc_out=.
	protoc proto/islb/islb.proto --go_opt=module=github.com/pion/ion --go_out=. --go-grpc_opt=module=github.com/pion/ion --go-grpc_out=.
	protoc proto/rtc/rtc.proto --go_opt=module=github.com/pion/ion --go_out=. --go-grpc_opt=module=github.com/pion/ion --go-grpc_out=.

proto_app:
	protoc apps/biz/proto/biz.proto --go_opt=module=github.com/pion/ion --go_out=. --go-grpc_opt=module=github.com/pion/ion --go-grpc_out=.
