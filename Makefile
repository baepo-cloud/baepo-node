DIRS := init nodeagent nodeagentctl

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

upgrade-proto-version:
	@echo "Please provide a commit hash for the upgrade"
	@echo "Usage: make upgrade-proto-version COMMIT=<commit-hash>"
ifdef COMMIT
	@echo "Upgrading to commit $(COMMIT) in all modules..."
	@for dir in $(DIRS); do \
		echo "Upgrading in $$dir..."; \
		cd $$dir && go get -u github.com/baepo-cloud/baepo-proto/go@$(COMMIT) && cd ..; \
	done
	@echo "All modules updated successfully!"
endif
