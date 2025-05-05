all: build-initcontainer build-init

.PHONY: all

build-initcontainer:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o resources/baepo-initcontainer -ldflags "-s -w" initcontainer/main.go

build-init:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o resources/baepo-init -ldflags "-s -w" init/main.go

build-node:
	go build -o resources/baepo-node -ldflags "-s -w" cmd/baepo-nodeserver/main.go

run-node: build-init build-initcontainer build-node
	./resources/baepo-node
