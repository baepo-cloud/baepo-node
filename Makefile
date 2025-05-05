all: build-initcontainer build-init

.PHONY: all

build-initcontainer:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o resources/baepo-initcontainer -ldflags "-s -w" initcontainer/main.go

build-init:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o resources/baepo-init -ldflags "-s -w" init/main.go

build-nodeagent:
	go build -o resources/baepo-nodeagent -ldflags "-s -w" nodeagent/main.go

run-node: build-init build-initcontainer build-nodeagent
	./resources/baepo-nodeagent
