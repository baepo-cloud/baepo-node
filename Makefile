all: build-initcontainer build-init

.PHONY: all tidy

build-initcontainer:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o resources/baepo-initcontainer -ldflags "-s -w" initcontainer/main.go

build-init:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o resources/baepo-init -ldflags "-s -w" init/main.go

build-nodeagent:
	go build -race -o resources/baepo-nodeagent -ldflags "-s -w" nodeagent/main.go

run-node: build-init build-initcontainer build-nodeagent
	./resources/baepo-nodeagent

upgrade-proto-version:
	@echo "Please provide a commit hash for the upgrade"
	@echo "Usage: make upgrade-proto-version COMMIT=<commit-hash>"
ifdef COMMIT
	@echo "Upgrading to commit $(COMMIT) in all modules..."
	@for dir in init nodeagent nodeagentctl vmruntime; do \
		echo "Upgrading in $$dir..."; \
		cd $$dir && go get -u github.com/baepo-cloud/baepo-proto/go@$(COMMIT) && cd ..; \
	done
	@echo "All modules updated successfully!"
endif

tidy:
	@echo "Tidying dependencies in all modules..."
	@find . -name "go.mod" -exec sh -c 'cd "$$(dirname {})" && echo "Tidying in $$(dirname {})" && go mod tidy' \;
	@echo "All modules tidied successfully!"
