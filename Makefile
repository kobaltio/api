build: # Build the application
	@go build -o main ./cmd/main.go

run: # Run the application
	@go run ./cmd/main.go

test: # Run tests
	@go test -v ./...

clean: # Clean the binary
	@rm -rf main

watch: # Live reload
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "installing air..."; \
		go install github.com/air-verse/air@latest; \
		air; \
	fi

lint: # Lint
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run ./...; \
	fi

help: # Print help
	@grep -E '^[a-zA-Z0-9 -]+:.*#'  Makefile | sort | while read -r l; do printf "\033[1;32m$$(echo $$l | cut -f 1 -d':')\033[00m:$$(echo $$l | cut -f 2- -d'#')\n"; done

.PHONY: build run test clean watch lint help