.PHONY: build test lint gocyclo golangci clean

build:
	@echo "Building armature-apprise..."
	@go build ./...

test:
	@echo "Running unit tests..."
	@go test -v ./...

lint: gocyclo golangci

gocyclo:
	@echo "Running gocyclo..."
	@if gocyclo -over 5 . | grep -qE "^[0-9]+"; then \
		gocyclo -over 5 .; \
		echo "Error: Functions with complexity > 5 found in library code"; \
		exit 1; \
	fi
	@echo "All library functions have complexity <= 5"

golangci:
	golangci-lint run

clean:
	@echo "Cleaning build artifacts..."
	@go clean ./...
