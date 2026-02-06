.PHONY: build test lint clean install

# Build binary
build:
	go build -o hy .

# Run all tests
test:
	go test ./cmd/... -v -race

# Run tests with coverage
test-cover:
	go test ./cmd/... -v -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run integration tests (requires HY_TEST_API_KEY)
test-integration:
	go test ./integration/... -v -tags=integration

# Lint code
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...
	goimports -w .

# Clean build artifacts
clean:
	rm -f hy coverage.out coverage.html

# Install locally
install: build
	cp hy ~/go/bin/

# Run all checks (CI simulation)
ci: fmt lint test build
	@echo "✓ All checks passed"

# Quick check before commit
pre-commit: fmt lint test
	@echo "✓ Ready to commit"
