#!/bin/bash
set -e

echo "🚀 Running Full Parevo Flow Test Suite..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Clean up
echo "🧹 Cleaning up..."
rm -f coverage.out benchmark.txt
go clean -testcache

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Step 1: Running unit tests with race detection..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

go test -v -race -coverprofile=coverage.out -covermode=atomic -timeout=60s

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 Step 2: Coverage Analysis..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Show detailed coverage
go tool cover -func=coverage.out

echo ""
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
COVERAGE_NUM=$(echo $COVERAGE | sed 's/%//')

echo "Total Coverage: $COVERAGE"

if (( $(echo "$COVERAGE_NUM < 80.0" | bc -l) )); then
    echo -e "${YELLOW}⚠️  Warning: Coverage ${COVERAGE}% is below 80% threshold${NC}"
else
    echo -e "${GREEN}✅ Coverage ${COVERAGE}% meets threshold${NC}"
fi

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html
echo "📄 HTML coverage report generated: coverage.html"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "⚡ Step 3: Running benchmarks..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

go test -bench=. -benchmem -run=^$ -timeout=5m | tee benchmark.txt

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🔍 Step 4: Running tests with count=10 (stability check)..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

go test -count=10 -timeout=5m -failfast

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🏗️  Step 5: Build check..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

go build -v ./...

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📦 Step 6: Go vet..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

go vet ./...

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📈 Summary:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ All unit tests passed"
echo "✅ Race detection: PASS"
echo "✅ Coverage: $COVERAGE"
echo "✅ Benchmarks completed"
echo "✅ Stability test (10x): PASS"
echo "✅ Build: SUCCESS"
echo "✅ Go vet: PASS"
echo ""
echo -e "${GREEN}🎉 Full test suite completed successfully!${NC}"
echo ""
echo "Generated files:"
echo "  - coverage.out (machine-readable coverage data)"
echo "  - coverage.html (HTML coverage report)"
echo "  - benchmark.txt (benchmark results)"
echo ""
