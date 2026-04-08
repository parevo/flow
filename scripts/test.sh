#!/bin/bash
set -e

echo "Running Core Test Suite..."
go test -v -short ./...
echo "Core tests completed successfully."
