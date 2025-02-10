GOFILES = $(shell find . -name \*.go)

build: # Build the application
	@echo "Building..."
	@go build -o main ./cmd/main.go

run: # Run the application
	@go run ./cmd/main.go

test: # Test the application
	@echo "Vetting..."
	@ go vet ./...
	@echo "Testing..."
	@go test ./... -v

clean: # Clean the binary
	@echo "Cleaning..."
	@rm -f main

watch: # Live reload
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "installing air..."; \
		go install github.com/air-verse/air@latest; \
		air; \
	fi

lint: # Run golangci-lint
	@if command -v golangci-lint > /dev/null; then \
		echo "Running golangci-lint..."; \
		golangci-lint run ./...; \
	else \
		echo "installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run ./...; \
	fi

fmt: # Run gofmt and goimports
	@echo "Verifying gofmt..."
	@!(gofmt -l -s -d ${GOFILES} | grep '[a-z]')

	@echo "Verifying goimports..."
	@!(go run golang.org/x/tools/cmd/goimports@latest -l -d ${GOFILES} | grep '[a-z]')

help: # Print help
	@grep -E '^[a-zA-Z0-9 -]+:.*#'  Makefile | sort | while read -r l; do printf "\033[1;32m$$(echo $$l | cut -f 1 -d':')\033[00m:$$(echo $$l | cut -f 2- -d'#')\n"; done

.PHONY: build run test clean watch verify fmt help
