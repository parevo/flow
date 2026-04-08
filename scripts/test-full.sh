#!/bin/bash
set -e

echo "Running Full Integration & Stress Test Suite..."
echo "---"
go test -v ./...
echo "---"
echo "All integration and stress tests completed successfully."
