#!/bin/bash

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

cd "$(dirname "${BASH_SOURCE[0]}")/proxy"

echo -e "${BLUE}=== Storage Test Suite ===${NC}\n"

echo -e "${YELLOW}[1/3] Deadlock Detection (Critical Bug Fix)${NC}"
go test -v -run TestStorageSync_NoDeadlock -timeout 5s
if [ $? -ne 0 ]; then
    echo -e "${RED}❌ CRITICAL: Sync() deadlock detected!${NC}"
    echo "Fix required: storage.go:81 should be 'defer s.mu.Unlock()'"
    exit 1
fi
echo -e "${GREEN}✅ No deadlock${NC}\n"

echo -e "${YELLOW}[2/3] Concurrency Safety (Race Detection)${NC}"
go test -race -run TestStorage -timeout 30s
if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Race conditions detected!${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Thread-safe${NC}\n"

echo -e "${YELLOW}[3/3] Functional Tests${NC}"
go test -v -run TestStorage
if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Functional tests failed${NC}"
    exit 1
fi
echo -e "${GREEN}✅ All functions working${NC}\n"

echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ Storage: All tests passed!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"