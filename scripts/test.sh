#!/bin/bash
set -e

echo "🧪 Running Parevo Flow Test Suite..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Run tests with coverage
go test -v -race -coverprofile=coverage.out -covermode=atomic -timeout=30s

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 Coverage Report:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Show coverage summary
go tool cover -func=coverage.out | tail -20

echo ""
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
echo "✅ Total Coverage: $COVERAGE"
echo ""
echo "🎉 All tests passed successfully!"
