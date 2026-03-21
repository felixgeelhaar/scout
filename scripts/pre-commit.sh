#!/usr/bin/env bash
# Pre-commit hook for browse-go
# Runs: gofmt, go vet, golangci-lint, unit tests, coverctl check, nox scan
#
# Install: ln -sf ../../scripts/pre-commit.sh .git/hooks/pre-commit

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

pass() { echo -e "${GREEN}✓${NC} $1"; }
fail() { echo -e "${RED}✗${NC} $1"; exit 1; }
warn() { echo -e "${YELLOW}!${NC} $1"; }

echo "browse-go pre-commit checks"
echo "==========================="

# 1. gofmt
echo -n "Checking gofmt... "
UNFMT=$(gofmt -l . 2>&1 | grep -v vendor | grep -v .git || true)
if [ -n "$UNFMT" ]; then
    fail "gofmt: these files need formatting:\n$UNFMT\nRun: gofmt -w ."
fi
pass "gofmt"

# 2. go vet
echo -n "Running go vet... "
if ! go vet ./... 2>&1; then
    fail "go vet found issues"
fi
pass "go vet"

# 3. golangci-lint (if available)
echo -n "Running golangci-lint... "
if command -v golangci-lint &>/dev/null; then
    if ! golangci-lint run --timeout 2m . ./cmd/... ./middleware/... ./internal/... 2>&1; then
        fail "golangci-lint found issues"
    fi
    pass "golangci-lint"
else
    warn "golangci-lint not installed, skipping"
fi

# 4. Unit tests (short mode, no integration tests)
echo -n "Running unit tests... "
if ! go test -short -race -count=1 ./... > /dev/null 2>&1; then
    fail "unit tests failed"
fi
pass "unit tests"

# 5. coverctl (if available)
echo -n "Running coverctl check... "
if command -v coverctl &>/dev/null; then
    if [ -f coverage.out ]; then
        if ! coverctl check --from-profile --profile coverage.out --config .coverctl.yaml > /dev/null 2>&1; then
            fail "coverctl: coverage below thresholds. Run 'make cover' first"
        fi
        pass "coverctl (from existing profile)"
    else
        warn "coverctl: no coverage.out found, run 'make cover' to generate"
    fi
else
    warn "coverctl not installed, skipping"
fi

# 6. nox scan (if available, via MCP only — skip in CLI pre-commit)
echo -n "Checking nox... "
if [ -f .nox/baseline.json ]; then
    pass "nox baseline present (run 'nox scan' via MCP for full check)"
else
    warn "no .nox/baseline.json found, run nox scan via MCP"
fi

echo ""
echo -e "${GREEN}All checks passed${NC}"
