#!/bin/bash

echo "=== Setting up test environment ==="

mkdir -p proxy/logs
mkdir -p backend/logs

if [ ! -f proxy/go.mod ]; then
    echo "Initializing proxy module..."
    cd proxy
    go mod init proxy 2>/dev/null || true
    go mod tidy
    cd ..
fi

if [ ! -f backend/go.mod ]; then
    echo "Initializing backend module..."
    cd backend
    go mod init backend 2>/dev/null || true
    go mod tidy
    cd ..
fi

chmod +x run_tests.sh
chmod +x test_day1.sh
chmod +x ci_test.sh

echo "✅ Test environment ready!"
echo ""
echo "Usage:"
echo "  ./test_day1.sh       # Run Day 1 tests"
echo "  ./run_tests.sh quick # Run all tests"
echo "  ./run_tests.sh help  # Show all options"