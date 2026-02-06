#!/bin/bash
# Install git pre-commit hook

HOOK_DIR=$(git rev-parse --git-dir)/hooks
PRE_COMMIT="$HOOK_DIR/pre-commit"

cat > "$PRE_COMMIT" << 'EOF'
#!/bin/bash
# Pre-commit hook: run tests and lint before commit

echo "Running pre-commit checks..."

# Format check
if ! go fmt ./... > /dev/null 2>&1; then
    echo "❌ go fmt failed"
    exit 1
fi

# Lint (if golangci-lint is installed)
if command -v golangci-lint &> /dev/null; then
    if ! golangci-lint run --fast; then
        echo "❌ Lint failed"
        exit 1
    fi
fi

# Tests
if ! go test ./cmd/... -count=1 > /dev/null 2>&1; then
    echo "❌ Tests failed"
    echo "Run 'go test ./cmd/... -v' for details"
    exit 1
fi

echo "✓ Pre-commit checks passed"
EOF

chmod +x "$PRE_COMMIT"
echo "✓ Pre-commit hook installed"
